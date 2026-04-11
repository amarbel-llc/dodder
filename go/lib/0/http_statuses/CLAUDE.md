# http_statuses

HTTP status code constants with formatted string representation.

## Type

- `Code`: HTTP status code with String() method formatting code and text

## Constants

- `Code400BadRequest`, `Code405MethodNotAllowed`
- `Code409Conflict`, `Code422UnprocessableEntity`
- `Code499ClientClosedRequest`: Custom nginx status code
- `Code500InternalServerError`, `Code501NotImplemented`

String() method formats as "code text" (e.g., "404 Not Found").
