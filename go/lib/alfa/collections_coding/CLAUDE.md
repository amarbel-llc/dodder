# collections_coding

Encoding utilities for collections with iterator support.

## Key Types

- `EncoderLike[T]` - interface for encoders that write to streams
- `EncoderJson[T]` - JSON encoder wrapper

## Key Functions

- `EncoderToWriter()` - converts EncoderLike to FuncIter for iteration-based writing
- `MakeEncoderJson()` - creates JSON encoder for type T

## Notes

- Integrates with interfaces.FuncIter for streaming collection writes
