# Feature Specification: APX-101 Docs/Code Parity Sweep

**Feature Branch**: `101-docs-code-parity`  
**Created**: March 8, 2026  
**Status**: Draft  

## Problem
The CLI documentation contains outdated "planned for a future release" notes marking commands as not yet available when those commands are already implemented and shipped. This drift makes the platform appear less mature than it actually is.

## Solution
Make documentation authoritative by auditing the actual command surface and removing "planned" notes for already-implemented features.

## User Scenarios & Testing

### User Story 1 - Remove outdated "planned" markers for implemented commands (Priority: P1)

A documentation reader or potential user consults the CLI reference and sees that `apx show`, `apx update`, and `apx upgrade` are available in the binary, but the docs say they're "planned for a future release." This creates confusion and doubt about platform maturity.

**Why this priority**: This directly impacts how users perceive APX. Fixing these notes prevents confusion and improves trust.

**Independent Test**: Can be fully tested by:
1. Running `apx --help` and verifying all listed commands exist
2. Checking docs/cli-reference/index.md contains no "planned" notes for actually-implemented commands
3. Verifying docs are consistent with actual command list from NewRootCmd()

**Acceptance Scenarios**:

1. **Given** user runs `apx --help`, **When** they read the output, **Then** they see all 20 registered subcommands
2. **Given** user consults docs/cli-reference/index.md, **When** they look for `apx show`, **Then** it is documented without "planned" markers
3. **Given** docs mention a command, **When** they claim it exists, **Then** that command is actually in NewRootCmd()

---

### User Story 3 - [Brief Title] (Priority: P3)

[Describe this user journey in plain language]

**Why this priority**: [Explain the value and why it has this priority level]

**Independent Test**: [Describe how this can be tested independently]

**Acceptance Scenarios**:

1. **Given** [initial state], **When** [action], **Then** [expected outcome]

---

[Add more user stories as needed, each with an assigned priority]

### Edge Cases

<!--
  ACTION REQUIRED: The content in this section represents placeholders.
  Fill them out with the right edge cases.
-->

- What happens when [boundary condition]?
- How does system handle [error scenario]?

## Requirements *(mandatory)*

<!--
  ACTION REQUIRED: The content in this section represents placeholders.
  Fill them out with the right functional requirements.
-->

### Functional Requirements

- **FR-001**: System MUST [specific capability, e.g., "allow users to create accounts"]
- **FR-002**: System MUST [specific capability, e.g., "validate email addresses"]  
- **FR-003**: Users MUST be able to [key interaction, e.g., "reset their password"]
- **FR-004**: System MUST [data requirement, e.g., "persist user preferences"]
- **FR-005**: System MUST [behavior, e.g., "log all security events"]

*Example of marking unclear requirements:*

- **FR-006**: System MUST authenticate users via [NEEDS CLARIFICATION: auth method not specified - email/password, SSO, OAuth?]
- **FR-007**: System MUST retain user data for [NEEDS CLARIFICATION: retention period not specified]

### Key Entities *(include if feature involves data)*

- **[Entity 1]**: [What it represents, key attributes without implementation]
- **[Entity 2]**: [What it represents, relationships to other entities]

## Success Criteria *(mandatory)*

<!--
  ACTION REQUIRED: Define measurable success criteria.
  These must be technology-agnostic and measurable.
-->

### Measurable Outcomes

- **SC-001**: [Measurable metric, e.g., "Users can complete account creation in under 2 minutes"]
- **SC-002**: [Measurable metric, e.g., "System handles 1000 concurrent users without degradation"]
- **SC-003**: [User satisfaction metric, e.g., "90% of users successfully complete primary task on first attempt"]
- **SC-004**: [Business metric, e.g., "Reduce support tickets related to [X] by 50%"]
