package ui

type cliTreeStateStackItem struct {
	parent     error
	child      error
	childIdx   int
	childCount int
}

func (stackItem cliTreeStateStackItem) isLastChild() bool {
	return stackItem.childIdx == stackItem.childCount-1
}

type cliTreeStateStack []cliTreeStateStackItem

func (stack cliTreeStateStack) len() int {
	return len(stack)
}

func (stack cliTreeStateStack) getDepth() int {
	return stack.len() - 1
}

// TODO refactor to remove parent and use stack instead
func (stack *cliTreeStateStack) push(
	parent error,
	child error,
) *cliTreeStateStackItem {
	stackItem := cliTreeStateStackItem{
		parent: parent,
		child:  child,
	}

	*stack = append(*stack, stackItem)

	return stack.getLast()
}

func (stack *cliTreeStateStack) pop() {
	*stack = (*stack)[:stack.getDepth()]
}

func (stack cliTreeStateStack) getLast() *cliTreeStateStackItem {
	return &stack[stack.getDepth()]
}
