# orgmode

Orgmode parser and serializer for dodder haustoria integration.

## Key Types

- `Document`: Parsed orgmode document with preamble and headings
- `Heading`: Single org heading with level, title, tags, properties, and body
- `Properties`: Key-value property drawer

## Key Functions

- `Parse`: Parse orgmode text into a Document
- `Serialize`: Render a Document back to orgmode text
- `MakeHeading`: Construct a heading from dodder object fields
