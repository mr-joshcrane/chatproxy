
## Testability

- [ ] Implementation can change without breaking tests (black-box tests)
- [ ] I can understand the expected behaviour of the system by reading tests
- [ ] Uses a variety of testing approaches for robustness (unit + integration etc)
- [ ] Has good code coverage and covers a good number of core and edge cases

## Maintainability

- [ ] Does the file structure follow a consistent pattern? (e.g. ports and adapters)
- [ ] Do file / function / variable / object names reflect their purpose?
- [ ] Is there any unnecessary coupling that would make refactoring or testing harder (e.g. email function and sms function linked together)?
- [ ] Is the code split into appropriate concerns / layers if necessary. E.g. API code does not interact with database.
- [ ] Is there any unnecessary complexity that makes the code harder to reason about?
- [ ] Comments provide extra context (the "why?") where necessary
- [ ] Is the API, architecture, setup and usage documented (e.g. README, OpenAPI, etc)?
- [ ] Is the code configurable? Can you change config in one place without re-factoring?

## Security and privacy

- [ ] Are there any opportunities for abuse? E.g. large volume of requests? Bad input?
- [ ] Are all entry points authenticated and authorised appropriately?
- [ ] Does any process, resource or user have more access than they need?
- [ ] Is all PII and sensitive data handled appropriately (not logged, not in plain text, not checked in)
- [ ] Are all third-party dependencies vetted and pinned?

## Robustness

- [ ] Can you think of any errors or conditions that would cause an unexpected state?
- [ ] Are there any statements that won't scale well to large data sets?
- [ ] Is there anything that can have an impact on system load / service limits / costs?
- [ ] Are all read and write operations logged to assist debugging and support?
- [ ] Are error messages readable and assist debugging and support (e.g. shows key non-sensitive details)?
