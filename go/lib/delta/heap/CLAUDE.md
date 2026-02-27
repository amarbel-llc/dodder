# heap

Generic heap implementation with pool integration and save/restore support.

## Key Types

- `Heap[E, E_PTR]`: Thread-safe generic heap with lessor comparison
- `Element`, `ElementPtr`: Heap element interfaces

## Features

- Push/pop with pool-managed elements
- Save/restore for iteration without destroying heap
- Lessor-based ordering
- Resetter integration for element reuse
- Thread-safe operations via mutex
