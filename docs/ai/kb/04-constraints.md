cocopilot/docs/ai/kb/04-constraints.md
```

```cocopilot/docs/ai/kb/04-constraints.md
# Constraints

This document outlines the assumptions and limitations that govern the design, implementation, and operation of the system. These constraints are critical to ensure the project remains aligned with its goals and adheres to the rules defined in the `.rules` file.

---

## 1. Technical Constraints
1.1 The system must be implemented in **Go** (Golang) as specified in the project requirements.
1.2 The database must use **SQLite** for task storage, as defined in the project architecture.
1.3 The server must run on **port 8080** by default (configurable via `COCO_HTTP_ADDR`) and support HTTP-based communication.
1.4 The system must be compatible with **Go 1.21 or later**.
1.5 The system must not introduce any new dependencies unless explicitly justified and approved.

---

## 2. Security Constraints
2.1 All inputs from users and agents must be validated to prevent injection attacks or other vulnerabilities.
2.2 No sensitive information (e.g., API keys, PII, secrets) should be logged or exposed in the system.
2.3 The system must follow the principle of **least privilege** for all operations and access control.
2.4 Default configurations must prioritize security (e.g., secure defaults for API endpoints).

---

## 3. Performance Constraints
3.1 The system must be optimized for handling a moderate number of concurrent agents and tasks without significant performance degradation.
3.2 No premature optimizations should be made; performance improvements must be based on measured bottlenecks.
3.3 The system must provide real-time updates for the Kanban UI using server-sent events (SSE).

---

## 4. Operational Constraints
4.1 The system must be deployable on a standard server environment without requiring specialized hardware or software.
4.2 The SQLite database file defaults to `./tasks.db` (configurable via `COCO_DB_PATH`) and should be easily reset by deleting the file.
4.3 The system must provide clear and actionable error messages for both users and agents.

---

## 5. Workflow Constraints
5.1 The project must adhere to the dual-voice workflow:
   - **Planner Phase**: Problem restatement, constraints, options, chosen path, verification hooks, and pre-mortem analysis.
   - **Executor Phase**: Minimal, reviewable changes with proper testing and adherence to the chosen plan.
5.2 All changes must include a summary, verification steps, and a confidence score.
5.3 The Knowledge Base (KB) must be updated for any changes to behavior, architecture, tradeoffs, assumptions, limitations, or risks.

---

## 6. Scope Constraints
6.1 The system is designed for orchestrating LLM agents and managing tasks. Features outside this scope must be explicitly approved before implementation.
6.2 The Kanban UI is limited to task management and does not include advanced analytics or reporting features.
6.3 The API is designed for task management and agent interaction only; additional functionalities must align with the project's goals.

---

## 7. Future Constraints
7.1 The system must remain extensible to support future features, but no speculative features should be implemented without clear requirements.
7.2 Any future changes must align with the project's goals and the constraints outlined in this document.

---

## 8. Documentation Constraints
8.1 All changes to the system that affect behavior or usage must be documented in the Knowledge Base (KB).
8.2 The README file must remain up-to-date with installation, usage, and API details.
8.3 Documentation must be clear, concise, and accessible to both developers and end-users.

---

## 9. Testing Constraints
9.1 All new features and bug fixes must include tests for:
   - Happy path scenarios.
   - At least one edge case.
   - Relevant failure modes.
9.2 Tests must be automated and runnable using standard Go testing tools.
9.3 Tests must not rely on external systems or services unless explicitly required.

---

## 10. Decision Constraints
10.1 All architectural and design decisions must be logged in `03-decisions.md`.
10.2 Revisiting decisions is allowed only if the prior decision is cited, changes are justified, and alternatives are presented.

---

These constraints are subject to revision as the project evolves. Any changes to this document must be logged in the Knowledge Base and approved through the established workflow.