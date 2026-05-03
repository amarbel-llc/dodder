# store_workspace

Workspace store interfaces for external object checkout and synchronization.

## Key Types

- `Supplies`: Dependencies bundle for workspace stores (dir, store, blob store, etc.)
- `StoreLike`: Main interface combining all workspace store capabilities
- `CheckoutOne`: Interface for checking out single objects
- `DeleteCheckedOut`: Interface for deleting checked out objects
- `MergeCheckedOut`: Interface for merging with conflict handling

## Key Interfaces

- `UpdateTransacted`: Updates transacted objects from external state
- `ReadCheckedOutFromTransacted`: Reads checkout state from internal objects
- `QueryCheckedOut`: Query interface for checked out objects
