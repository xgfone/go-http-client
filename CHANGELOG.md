
## v0.19.0

Refactor the codes.

- **New**:
    - Add the `BaseURL` and `Do` methods for `Client`.
- **Remove**:
    - Remove the header constants.
    - Remove the useless type `Decoder`.
    - Remove the `ToError` method from `Response`.
    - Remove the functions `XxxJSON`, `CloseBody`, `GetContentType` and `NewRequestWithContext`.
- **Changed**:
    - The `Hook` interface also returns an error.
    - Use the `Doer` interface instead of the `*http.Client` type.
    - Remove the argument `code` from the function `NewError`.
