package regex

type Regex *nfaState

func Compile(str string) Regex {
	return nfa(postfix(preprocess(str)))
}

func Match(regex Regex, str string) bool {
	return match(regex, str)
}

// private

const (
	opGroupStart = '('
	opGroupEnd   = ')'
	opAnd        = '&'
	opOr         = '|'
)

var operators = [...]rune{opGroupStart, opGroupEnd, opAnd, opOr}

// The higher the index, the higher the precedence
var operatorPrecedence = [...]rune{opOr, opAnd}

func isOperator(char rune) bool {
	for _, op := range operators {
		if op == char {
			return true
		}
	}
	return false
}

func precedence(char rune) int {
	result := -1

	for index, op := range operatorPrecedence {
		if char == op {
			result = index
		}
	}

	return result
}

// Inserts opAnd in the source string
func preprocess(str string) string {
	result := make([]rune, 0, 1024)

	// Count of groups we are currently nested in
	groupStackCounter := 0
	// Previous char expects to be connected to the next one with opAnd
	wantsAnd := false
	// Previous char allows the current char to be opOr
	canOr := false
	// Previous char was opOr
	lastOr := false

	for _, char := range str {
		switch char {
		case opGroupStart:
			// Group
			groupStackCounter = groupStackCounter + 1

			// And
			if wantsAnd {
				result = append(result, opAnd)
				wantsAnd = false
			}

			// Or
			canOr = false
			lastOr = false

			// Result
			result = append(result, char)
		case opGroupEnd:
			// Group
			groupStackCounter = groupStackCounter - 1

			// And
			wantsAnd = true

			// Or
			canOr = true
			if lastOr {
				panic("or misused")
			}
			lastOr = false

			// Result
			result = append(result, char)
		case opOr:
			// And
			wantsAnd = false

			// Or
			if !canOr {
				panic("or misused")
			}
			canOr = false
			lastOr = true

			// Result
			result = append(result, char)
		default:
			// And
			if wantsAnd {
				result = append(result, opAnd)
			}
			wantsAnd = true

			// Or
			canOr = true
			lastOr = false

			// Result
			result = append(result, char)
		}
	}

	if groupStackCounter != 0 {
		panic("group mismatch")
	}

	return string(result)
}

type operatorStack struct{ buf []rune }

func newOperatorStack() operatorStack {
	buf := make([]rune, 0, 1024)
	return operatorStack{buf: buf}
}
func (stack *operatorStack) Empty() bool {
	return len(stack.buf) == 0
}
func (stack *operatorStack) Peek() rune {
	return stack.buf[len(stack.buf)-1]
}
func (stack *operatorStack) Push(op rune) {
	stack.buf = append(stack.buf, op)
}
func (stack *operatorStack) Pop() rune {
	last := stack.buf[len(stack.buf)-1]
	stack.buf = stack.buf[:len(stack.buf)-1]
	return last
}

// Converts an infix notation into a postfix notation
func postfix(str string) string {
	result := make([]rune, 0, 1024)
	operatorStack := newOperatorStack()

	for _, char := range str {
		if isOperator(char) {
			switch char {
			case opGroupStart:
				// Group start lands on the stack and waits there for group end
				operatorStack.Push(char)
			case opGroupEnd:
				// Group end pops all the operators from the stack until it meets a group start
				for {
					if operatorStack.Peek() == opGroupStart {
						operatorStack.Pop()
						break
					} else {
						result = append(result, operatorStack.Pop())
					}
				}
			default:
				// Other operators pop all the higher or equal precedence operators from the stack
				// before going on the stack.
				for {
					if operatorStack.Empty() {
						operatorStack.Push(char)
						break
					} else if precedence(operatorStack.Peek()) >= precedence(char) {
						result = append(result, operatorStack.Pop())
					} else {
						operatorStack.Push(char)
						break
					}
				}
			}
		} else {
			// Regular char is just appended
			result = append(result, char)
		}
	}

	for {
		if operatorStack.Empty() {
			break
		} else {
			result = append(result, operatorStack.Pop())
		}
	}

	return string(result)
}

const (
	nfaStateKindChar int = iota
	nfaStateKindSplit
	nfaStateKindMatch
)

// Each state in the automaton is either a match, a single matching char leading to the
// next state, or a split leading to two states.
type nfaState struct {
	char rune
	kind int
	out1 *nfaState
	out2 *nfaState
}

// A fragment is a part of the state graph. It has a start and lists of output arrows.
// Each arrow may not yet point at anything.
type nfaFrag struct {
	start *nfaState
	outs  []**nfaState
}

// Hooks up the dangling arrows of a fragment to the given state.
func connectNfaFrag(frag *nfaFrag, state *nfaState) {
	for _, out := range frag.outs {
		*out = state
	}
}

type nfaFragStack struct{ buf []*nfaFrag }

func newNfaFragStack() nfaFragStack {
	buf := make([]*nfaFrag, 0, 1024)
	return nfaFragStack{buf: buf}
}
func (stack *nfaFragStack) Push(frag *nfaFrag) {
	stack.buf = append(stack.buf, frag)
}
func (stack *nfaFragStack) Pop() *nfaFrag {
	last := stack.buf[len(stack.buf)-1]
	stack.buf = stack.buf[:len(stack.buf)-1]
	return last
}

// Creates a set of interconnnected states that make up an NFA for matching
// the given regex in postfix form
func nfa(str string) *nfaState {
	fragStack := newNfaFragStack()

	for _, char := range str {
		switch char {
		case opAnd:
			// opAnd connects the last two fragments
			frag2 := fragStack.Pop()
			frag1 := fragStack.Pop()
			connectNfaFrag(frag1, frag2.start)
			frag := &nfaFrag{start: frag1.start, outs: frag2.outs}
			fragStack.Push(frag)
		case opOr:
			// opOr makes a split into the last two fragments
			frag2 := fragStack.Pop()
			frag1 := fragStack.Pop()
			state := &nfaState{kind: nfaStateKindSplit, out1: frag1.start, out2: frag2.start}
			frag := &nfaFrag{start: state, outs: append(frag1.outs, frag2.outs...)}
			fragStack.Push(frag)
		default:
			// regular non-op char adds a new state with one dangling arrow
			state := &nfaState{char: char, kind: nfaStateKindChar}
			frag := &nfaFrag{start: state, outs: []**nfaState{&state.out1}}
			fragStack.Push(frag)
		}
	}

	// All the dangling arrows of the final fragment are connected to the match state
	result := fragStack.Pop()
	match := &nfaState{kind: nfaStateKindMatch}
	connectNfaFrag(result, match)

	return result.start
}

// Adds the state to the list of states.
// If the state is a split, recursively adds its children.
func appendState(stateList []*nfaState, state *nfaState) []*nfaState {
	if state.kind == nfaStateKindSplit {
		return appendState(appendState(stateList, state.out1), state.out2)
	} else {
		return append(stateList, state)
	}
}

// Uses the given nfa to match a string.
func match(nfa *nfaState, str string) bool {
	currentStates := make([]*nfaState, 0, 1024)
	nextStates := make([]*nfaState, 0, 1024)

	currentStates = appendState(currentStates, nfa)

	for _, char := range str {
		for _, state := range currentStates {
			if state.char == char {
				nextStates = appendState(nextStates, state.out1)
			}
		}

		currentStates = nextStates
		nextStates = nextStates[:0]
	}

	for _, state := range currentStates {
		if state.kind == nfaStateKindMatch {
			return true
		}
	}

	return false
}
