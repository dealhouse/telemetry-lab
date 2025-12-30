# ADR 0001: Polyglot tech stack & repo structure

## Status
Accepted

## Context
Goal is language/framework practice (Go, C++, Python, Java) plus SDLC artifacts (CI, UML, docs).
Want a structure that supports independent components while staying simple early.
Avoiding Docker initially to keep local iteration light.

## Decision
Use a mono-repo with language-specific components:
- Go for ingestion gateway
- C++ for parsing/rules engine
- Python for analytics jobs
- Java/Spring Boot for a read API (OpenAPI)
UI will be added later.

Documentation will live in `/docs` with:
- ADRs in `/docs/adr`
- UML diagrams in `/docs/diagrams` (PlantUML)

## Alternatives Considered
- Separate repos per service (more overhead, harder to coordinate)
- Single-language implementation (doesn't meet practice goal)

## Consequences
+ Easy to show SDLC maturity in one place
+ Clear boundaries when we start wiring components
- Requires discipline to keep each component well-scoped
