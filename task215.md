This is already useful, but right now it feels like an internal control panel instead of a real mission console.

The biggest problem is not that it lacks pages. The biggest problem is that the UI exposes less than the backend already knows.

What’s clearly true from the repo
	•	The main Kanban UI is mostly a single inline page in server/ui_pages.go.
	•	The board is still centered on a simple 3-column flow and a minimal create-task modal.
	•	Your data model already supports richer task metadata like title, type, priority, tags, approval status, parent task, etc. in internal/models/models.go.
	•	You already have richer project/backend capabilities like a project dashboard endpoint and task templates, but the homepage doesn’t really surface them.
	•	There are also a couple obvious UI polish issues in the current page code, like CSS vars being used without being defined (--vscode-dropdown-bg, --vscode-charts-*).

What I’d change first

1) Stop hiding important task metadata

Right now task creation is too thin.

The create modal should support:
	•	title
	•	type
	•	priority
	•	tags
	•	parent
	•	dependencies
	•	template picker
	•	requires approval

Why this matters:
	•	Your backend already understands richer task state.
	•	Humans and agents both move faster when tasks are classified at creation time.
	•	“Instructions only” scales badly once the queue gets dense.

Default move: switch task creation to the v2 task-create flow and make the modal a real task composer, not just a text box.

⸻

2) Add a right-side task detail drawer

Expanding cards inline is okay for 5 tasks. It gets messy fast.

Each task should open a detail drawer with:
	•	full instructions
	•	output
	•	status history
	•	dependencies
	•	linked run(s)
	•	artifacts / logs
	•	approval state
	•	quick actions: edit, duplicate, block, fail, approve, reopen

Why this matters:
	•	It keeps the board scannable.
	•	It turns the board into triage + execution instead of “expand/collapse archaeology.”

⸻

3) Upgrade the board from “3 buckets” to “operational states”

The current To Do / In Progress / Done model is too lossy for an agentic system.

Your UI should either:
	•	switch to v2 statuses directly, or
	•	keep 3 high-level buckets but show substate badges like:
	•	queued
	•	claimed
	•	running
	•	needs review
	•	failed
	•	cancelled
	•	blocked

Why this matters:
	•	Claimed vs running vs review are materially different.
	•	Operators need to know whether work is merely assigned, actively executing, or waiting on humans.

⸻

4) Add board controls that operators actually need

You need a control strip above the board with:
	•	search
	•	filter by agent
	•	filter by type
	•	filter by priority
	•	filter by project tag
	•	show blocked only
	•	sort by priority / age / updated time

Right now the board works as a passive list. It needs to become a queryable operating surface.

⸻

5) Make the homepage a dashboard, not just a board

The board should be a view, not the whole product identity.

Your default project page should show:
	•	queue counts by status
	•	stalled tasks
	•	active runs
	•	recent failures
	•	agent heartbeat summary
	•	automation events
	•	recommendations
	•	recent repo changes

You already appear to have backend support for dashboard-style data. Surface it.

Best structure:
	•	/ → dashboard
	•	/board → Kanban
	•	/runs → execution
	•	/agents → workforce
	•	/repo → code context
	•	/audit → traceability

⸻

6) Fix the header information architecture

The top nav is overloaded.

Right now it reads like “every endpoint is a tab.”

Change it to:
	•	Primary nav: Dashboard, Board, Runs, Agents, Repo
	•	Secondary/admin: Policies, Memory, Context Packs, Audit, Settings, Health

Also move workdir out of the main header. That is advanced config, not top-level daily UX.

⸻

7) Make agent status actually useful

The current agent dropdown is too thin.

Each agent row should show:
	•	online/offline
	•	last heartbeat
	•	current task
	•	current run id
	•	lease expiry
	•	recent completion count
	•	failure count in last N hours

Also add visual alerts for:
	•	agent offline with claimed task
	•	agent stale lease
	•	repeated task failures
	•	queue backing up while agent count is low

That turns “Agents: 0 online” into something operational.

⸻

8) Surface dependencies directly on cards

You have a DAG page, but the main board should show dependency state without leaving context.

Each card should show:
	•	blocked/unblocked badge
	•	dependency count
	•	parent/child count
	•	“waiting on #123” quick link

This matters because blocked tasks should visually disappear from “ready to act” mental space.

⸻

9) Improve task cards for scan speed

Current cards are readable but low-density.

Each card should show, at a glance:
	•	task ID
	•	short title
	•	type badge
	•	priority badge
	•	agent badge
	•	blocked/approval badge
	•	relative updated time
	•	small output/result summary

The current design spends too much space on expansion mechanics and not enough on signal.

⸻

10) Fix the obvious polish debt

These are small, fast wins:
	•	define missing CSS variables
	•	replace alert() / confirm() with proper toast + modal UI
	•	add loading / reconnect indicators for SSE
	•	show empty states with action buttons
	•	persist expanded state / filters in local storage
	•	add keyboard shortcuts:
	•	n new task
	•	/ search
	•	g b board
	•	g r runs

These are cheap and make the product feel way more serious.

What I would not do yet

Do not jump straight to:
	•	a full React rewrite
	•	multi-user SaaS auth complexity
	•	mobile-first redesign
	•	fancy animations
	•	plugin marketplace ideas

That is drift.

Your bottleneck is operator speed and visibility, not framework prestige.

Best next build order

Phase 1 — fast wins
	1.	Fix CSS token bugs and add proper toast/confirm UI
	2.	Add search/filter/sort bar above the board
	3.	Add priority/type/status badges to task cards
	4.	Add blocked/dependency indicators

Phase 2 — leverage
	5.	Build the task detail drawer
	6.	Expand task creation to title/type/priority/tags/template/dependencies
	7.	Show agent lease + heartbeat + current task

Phase 3 — identity shift
	8.	Add dashboard homepage using existing backend summary data
	9.	Split server/ui_pages.go into reusable templates/static assets
	10.	Move board to richer v2 task states

The highest-leverage single change

If you only do one thing next:

Build the task detail drawer + richer task creation modal.

That solves the main weakness:
	•	creation is too weak
	•	inspection is too clunky
	•	the board is overloaded because details are trapped inside cards

Bottom line

The product is already valid.
What it needs now is less “proof-of-concept board” and more “operator cockpit.”

Your next goal is not more endpoints.
Your next goal is to make the existing system’s intelligence visible, sortable, and actionable.

Next move should be turning this into a concrete repo backlog of 8–12 implementation tasks in Cocopilot’s own task format.
Yes — and I’d do it without throwing away this repo.

What’s known

From the repo shape:
	•	this is a Go app with a local SQLite core
	•	the current UI is mostly server-rendered HTML in server/ui_pages.go
	•	the backend already has a richer model than the UI exposes:
	•	v2 task states
	•	task type
	•	priority
	•	tags
	•	parent task
	•	approval status
	•	templates
	•	dashboard data
	•	so the real problem is presentation and input ergonomics, not missing backend foundations

My recommendation

Default path: build a native desktop operator app and keep the Go backend as the control plane.

Do not rewrite the task engine first.
Do not move logic into the GUI.
Do not start by replacing APIs.

Best architecture decision

There are 3 realistic paths:

Option A — macOS-first native app with SwiftUI + Go core

Best if your real goal is an actual GUI, polished, fast, native-feeling.
	•	SwiftUI desktop app for windows, panes, forms, tables, inspectors
	•	current Go service remains the source of truth
	•	app talks to Go over localhost or Unix socket
	•	later you can keep web UI as fallback/admin surface

Pros
	•	real native UX
	•	best tables/forms/navigation on Mac
	•	easiest way to make it feel like a serious operator console

Cons
	•	two languages
	•	more integration work

Option B — all-Go native app with Fyne or Gio

Best if your goal is ship fast in one language.
	•	same repo
	•	less platform polish
	•	easier reuse by current codebase

Pros
	•	one language
	•	fastest route to “not web”
	•	easiest staffing fit for this repo

Cons
	•	UI will likely feel less premium
	•	complex desktop workflows are clunkier than SwiftUI

Option C — Wails/Tauri

I would not recommend this as the main answer to your request.

Why:
	•	it is still fundamentally web-tech UI in a desktop shell
	•	you specifically asked for actual GUI instead of web UI

Default recommendation

Because you’re on Mac and you want a real GUI:

Pick Option A: SwiftUI shell + Go backend.

That gives you the cleanest result and preserves the repo’s existing strength.

⸻

Plan

Phase 0 — lock the product decision

Before coding, decide this explicitly:

Cocopilot becomes a local desktop control room for agent orchestration.

That means the GUI must optimize for:
	•	task triage
	•	execution monitoring
	•	dependency management
	•	run inspection
	•	approval flows
	•	project context
	•	fast, safe task creation

Not “browse pages.”

⸻

Phase 1 — separate core logic from web presentation

Right now the web UI and route handlers are too close to the operator flow.

Goal

Create a service layer that both the web UI and the desktop app can use.

Work

Refactor these areas:
	•	task service
	•	project service
	•	run service
	•	agent service
	•	dashboard service
	•	template service

Rule

Business logic should not live in:
	•	HTML builders
	•	route handlers
	•	UI-specific request parsing

Target shape

Create something like:
	•	internal/service/tasks
	•	internal/service/projects
	•	internal/service/runs
	•	internal/service/agents
	•	internal/service/dashboard

Then:
	•	HTTP handlers call service layer
	•	desktop app calls same logic through API or adapter

This is the main architectural unlock.

⸻

Phase 2 — define a GUI-first information architecture

The current nav is endpoint-shaped. The desktop app should be operator-shaped.

Main app sections

1. Dashboard

Shows:
	•	queued / running / failed / review counts
	•	stalled tasks
	•	active runs
	•	recent failures
	•	recent automation events
	•	recommendations
	•	agent health

2. Tasks

Main working surface:
	•	board view
	•	table view
	•	dependency view
	•	task detail inspector

3. Runs

For execution visibility:
	•	live logs
	•	step timeline
	•	artifacts
	•	tool invocations
	•	failure details

4. Agents

For workforce visibility:
	•	online/offline
	•	busy/idle
	•	last seen
	•	claimed task
	•	stale lease warnings

5. Projects

For workspace context:
	•	workdir
	•	templates
	•	policies
	•	memory
	•	repo snapshot
	•	automation settings

6. Settings / Admin

For:
	•	health
	•	backup/restore
	•	auth
	•	config
	•	migrations
	•	diagnostics

⸻

Phase 3 — replace free-text fields with typed controls

This is one of the highest-leverage fixes.

You’re right: if the system knows the expected type, the UI should not force typing.

Build a field-control registry

Create a field schema that maps field type → UI control.

Examples:
	•	status → segmented control / dropdown
	•	task type → dropdown
	•	priority → stepper or segmented control
	•	project → searchable dropdown
	•	parent task → searchable task picker
	•	dependencies → multi-select task picker
	•	tags → chips with autocomplete
	•	requires approval → toggle
	•	approval status → read-only badge or review control
	•	agent → searchable dropdown
	•	workdir → file/folder picker
	•	limit → numeric input with bounds
	•	dates → date/time picker
	•	paths → path picker
	•	booleans → toggle
	•	enum policies → dropdown
	•	JSON blobs → advanced editor, not default field

Immediate places to apply this

Task creation

Replace the current modal with:
	•	title
	•	instructions
	•	type
	•	priority
	•	tags
	•	project
	•	parent task
	•	dependencies
	•	requires approval
	•	template selector

Filters

Replace text filter junk with:
	•	status dropdown
	•	type dropdown
	•	agent dropdown
	•	priority filter
	•	project dropdown
	•	blocked toggle
	•	sort selector

Settings/config

Use:
	•	toggles
	•	steppers
	•	enum selectors
	•	file pickers

Not raw text.

⸻

Phase 4 — build the actual desktop screens

Do this in this order.

Screen 1: Task board + inspector

This is the first screen because it replaces the current homepage pain.

Left/main pane
	•	board columns or grouped list
	•	search
	•	filters
	•	sort
	•	keyboard navigation

Right pane inspector

When a task is selected, show:
	•	title
	•	full instructions
	•	output
	•	status
	•	type
	•	priority
	•	parent
	•	dependencies
	•	approval
	•	related runs
	•	timestamps
	•	actions

Actions
	•	claim
	•	start
	•	mark review
	•	complete
	•	fail
	•	cancel
	•	duplicate
	•	create child task
	•	add dependency

This alone makes the product feel 10x better.

⸻

Screen 2: Task creation wizard

Do not use a single giant form first.

Use a 3-step wizard:

Step 1 — task intent
	•	template
	•	title
	•	instructions
	•	type

Step 2 — execution metadata
	•	priority
	•	tags
	•	parent
	•	dependencies
	•	approval requirement

Step 3 — review
	•	summary
	•	validation
	•	create

This reduces bad task creation.

⸻

Screen 3: Runs viewer

This should feel like an IDE debug panel.

Include:
	•	live logs
	•	step timeline
	•	artifacts list
	•	tool invocations
	•	structured metadata
	•	failure summary
	•	rerun / inspect source task

⸻

Screen 4: Agents console

This should answer in 3 seconds:
	•	who is alive
	•	who is stale
	•	who is stuck
	•	what each agent is doing
	•	what failed recently

Add:
	•	heartbeat age
	•	current run
	•	current task
	•	recent completions
	•	recent failures
	•	lease warnings

⸻

Screen 5: Project dashboard

Use the backend dashboard data you already have.

Show:
	•	key counts
	•	stalled tasks
	•	active runs
	•	recent failures
	•	repo changes
	•	recommendations

This should be the app landing page, not raw kanban.

⸻

Phase 5 — add native desktop behaviors

This is where a GUI starts beating the web UI.

Add:
	•	global command palette
	•	keyboard shortcuts
	•	native notifications
	•	drag-and-drop task ordering
	•	split panes
	•	resizable inspectors
	•	context menus
	•	multi-select bulk actions
	•	quick-add sheet
	•	file chooser dialogs
	•	tray/menu bar status
	•	local reopen/recent projects
	•	window state persistence

These are native wins. Use them.

⸻

What else I think you should change

Beyond dropdowns and native windows, these are the important upgrades.

1. Templates should drive task creation

You already have template support in the backend. Surface it hard.

Every common task should start from:
	•	template
	•	prefilled type
	•	prefilled priority
	•	prefilled tags
	•	prefilled approval rules

That removes garbage input.

2. Add dependency-aware picking

Do not make users type task IDs.

Use:
	•	searchable task picker
	•	dependency graph preview
	•	blocked/unblocked indicator

3. Add validation before submit

Examples:
	•	parent must be same project
	•	dependency cannot point to self
	•	review-required tasks cannot auto-complete
	•	title required for non-template tasks
	•	priority range enforced by UI

4. Add “simple mode” and “power mode”

Some screens should collapse complexity by default.

Simple mode
	•	title
	•	instructions
	•	template
	•	priority

Power mode
	•	type
	•	tags
	•	dependencies
	•	parent
	•	approval
	•	automation depth
	•	advanced metadata

5. Reduce raw JSON exposure

JSON should exist for power users, but it should not be the primary interface.

Use:
	•	property panels
	•	badges
	•	pills
	•	tables
	•	structured sections

6. Add inline approval workflow

Since approval is already in the model, the GUI should support:
	•	approve
	•	reject
	•	show reason
	•	show pending approval queue

7. Add richer tables everywhere

For tasks, runs, agents, events:
	•	sortable columns
	•	saved filters
	•	column visibility
	•	multi-select
	•	quick actions

8. Add a repo-aware picker

Since this system is code-oriented, GUI should let you:
	•	attach files to task context
	•	choose repo paths from a tree
	•	preview changed files
	•	create task from selected files

That is a huge operator win.

⸻

Biggest risk

The biggest failure mode is this:

you accidentally build a second product instead of a new interface.

Avoid that by keeping these rules:
	•	Go backend remains source of truth
	•	desktop app is a new presentation layer
	•	no duplicated task logic in the client
	•	service contracts stay stable
	•	web UI remains as fallback until parity is reached

⸻

Recommended implementation order

Track 1 — architecture
	1.	extract service layer from route/UI code
	2.	define desktop-facing view models
	3.	define field schema / typed input metadata

Track 2 — app shell
	4.	create desktop app shell
	5.	sidebar + routing + window layout
	6.	dashboard screen

Track 3 — task workflow
	7.	task board/table
	8.	task detail inspector
	9.	task creation wizard with selectors

Track 4 — execution workflow
	10.	runs viewer
	11.	agents console
	12.	notifications + keyboard shortcuts

Track 5 — cleanup
	13.	web/desktop parity audit
	14.	remove duplicate logic from web handlers
	15.	keep web UI as backup/admin surface

⸻

The single smartest move first

Before any GUI coding:

build a typed field schema and service layer boundary.

Because once that exists:
	•	dropdowns become easy
	•	validation becomes consistent
	•	both web and desktop can share behavior
	•	future forms stop becoming custom hacks

Without that, you’ll just rebuild the current mess in a native window.

⸻

Bottom line

Your instinct is right:
	•	yes, move toward a real GUI
	•	yes, replace known-value text inputs with selectors
	•	yes, stop making the operator type IDs and enum values manually

My default recommendation is:

macOS-first SwiftUI desktop app on top of the existing Go control plane, with typed forms, task inspector panes, and dashboard-first navigation.

The next practical step is turning this into a repo execution plan with concrete engineering tasks in build order.
Yes. The issue is bigger than “fix one page.”

You have a product-flow mismatch:
	•	the repo is public
	•	the README/instructions create a “copy text into agent chat and let it curl localhost” workflow
	•	but that workflow bypasses key platform features
	•	so the system looks less capable than it is
	•	and some pages are genuinely broken on top of that

What’s actually going wrong

1) The usage path does not produce a visible agent

Your current agent flow is effectively:
	1.	user starts server
	2.	user copies initial instructions
	3.	agent polls tasks
	4.	agent works

But the agent UI is backed by the agents table, and the claim path does not register/update an agent automatically.

So you get this failure mode:
	•	tasks can be claimed with an agent_id
	•	runs and leases can exist
	•	but /api/v2/agents still only shows agent_default or stale entries
	•	therefore the user thinks the agent system is broken

That is not a small UX bug. That is a core product contradiction.

⸻

2) The onboarding instructions are teaching the wrong path

Your generated instructions still point the agent toward a curl-driven loop, and one of the example paths is wrong:
	•	getInstructions() references /api/v2/projects/default/tasks/claim-next
	•	your actual default project elsewhere is proj_default

That alone can derail a first-run experience.

Worse: the instructions teach “use curl and keep polling,” which reduces Cocopilot to a queue endpoint instead of making the agent use:
	•	registration
	•	heartbeats
	•	runs
	•	step logging
	•	artifacts
	•	context packs
	•	richer task metadata

So the platform’s strongest features are present but not being exercised.

⸻

3) The reference worker is stale against the current API shape

The built-in worker is a serious red flag.

It posts to claim-next, but then it expects an older response format like:
	•	envelope
	•	or task_id

Your current claim-next handler returns:
	•	task
	•	lease
	•	run
	•	context

So the reference worker path is not aligned with the current contract.

That means your own “official path” is not a reliable proof of product quality.

⸻

4) Some pages are actually broken, not just weak

I found at least these concrete mismatches:

Events page route bug
	•	nav links to /events-browser
	•	handler checks for /events
	•	result: broken page

Context packs page contract bug
	•	UI placeholder suggests task ID like task_abc123
	•	backend expects task_id as a positive integer

Context pack detail display bug
	•	detail page looks for contents.files
	•	context-pack creation currently stores file metadata under contents.repo_files.files

So even when data exists, the UI can look empty or wrong.

Runs page is not a real page
It is mostly a manual run-ID lookup surface. That is not enough for a public product. A user cannot discover runs from it.

⸻

What the product should become

You need to redefine the default usage model.

New default model

Today’s mental model

“Cocopilot is a localhost task server that an LLM can poll with curl.”

Better mental model

“Cocopilot is a local operator app and agent runtime.
You launch it, connect agents, watch work happen, inspect runs, and manage tasks from one place.”

That means the product should own:
	•	onboarding
	•	agent registration
	•	run visibility
	•	context assembly
	•	task creation
	•	execution telemetry
	•	recovery when agents go stale

Not leave those to ad hoc prompt paste.

⸻

My recommendation

Strategic direction

Do this as one coordinated upgrade with two tracks running together:

Track A — stabilize and make the current product trustworthy

Fix every public-facing broken path now.

Track B — transition from “web admin toy” to “desktop operator platform”

Move toward a real GUI and agent-aware onboarding.

Do not do GUI first while the core flow is inconsistent.
Do not only patch bugs while leaving the onboarding model broken.

Both have to move together.

⸻

The correct product flow

This should be the canonical experience.

First-run operator flow
	1.	user downloads/clones repo
	2.	user launches Cocopilot
	3.	app creates/opens project
	4.	app shows dashboard
	5.	app offers:
	•	“Connect built-in worker”
	•	“Connect external agent”
	•	“Import existing project”
	6.	user chooses agent mode
	7.	Cocopilot generates a bootstrap payload/script
	8.	agent registers itself
	9.	agent heartbeats
	10.	agent claims tasks
	11.	agent logs run steps/logs/artifacts
	12.	UI reflects all of it live

That is the product.

⸻

What to build

Phase 1 — emergency public-repo stabilization

This is the “make it not embarrassing” pass.

Must-fix immediately
	1.	Fix events page route
	•	/events-browser handler must actually serve /events-browser
	2.	Fix initial instructions
	•	replace default with proj_default
	•	remove stale curl-only guidance
	•	teach the proper lifecycle:
	•	register agent
	•	claim
	•	heartbeat
	•	log run progress
	•	complete task
	3.	Fix built-in worker
	•	update it to parse current claim-next response
	•	make it consume {task, lease, run, context}
	•	add lease heartbeat support
	•	add run step/log/artifact reporting
	4.	Fix context packs page
	•	task ID field must be numeric
	•	task picker should be dropdown/search, not raw text
	•	detail page must render repo_files.files correctly
	5.	Fix runs page
	•	add recent runs list
	•	stop requiring manual run ID entry as the only path
	6.	Do a full public-page audit
	•	dashboard
	•	kanban
	•	projects
	•	agents
	•	runs
	•	memory
	•	policies
	•	dependencies
	•	context packs
	•	events
	•	graphs
	•	repo
	•	audit
	•	settings
	•	health

Every page needs:
	•	valid route
	•	working data load
	•	non-empty empty state
	•	no stale API contract assumptions

⸻

Phase 2 — fix the agent lifecycle model

This is the real heart of the upgrade.

Goal

An agent should become visible and healthy without extra ceremony.

Change 1: agent auto-upsert on claim

When claim-next or claim-by-id receives agent_id:
	•	if agent exists → heartbeat/update last_seen
	•	if agent does not exist → create it automatically with minimal profile

This should happen server-side.

Why:
	•	it removes the current blind spot
	•	it makes the system resilient even if the client is simple

Change 2: explicit registration still exists

Keep POST /api/v2/agents, but use it for richer metadata:
	•	name
	•	capabilities
	•	metadata
	•	version
	•	transport mode
	•	host info

Change 3: heartbeat becomes standard

Agents should periodically call:
	•	/api/v2/agents/{id}/heartbeat
	•	and /api/v2/leases/{leaseId}/heartbeat when working a task

Change 4: server should infer health

Agent status should derive from:
	•	registered_at
	•	last_seen
	•	current lease
	•	current run
	•	completion recency

Then UI can show:
	•	online
	•	idle
	•	busy
	•	stale
	•	offline

⸻

Phase 3 — replace prompt-paste onboarding with actual onboarding

Right now the “copy initial instructions” model is clever, but too weak as the primary flow.

Better approach

Give the user a Connect Agent action.

For external chat agents

Generate a structured bootstrap package:
	•	server URL
	•	project ID
	•	agent ID
	•	registration payload
	•	claim loop rules
	•	heartbeat rules
	•	completion contract
	•	examples

For local worker

Provide:
	•	one-click start
	•	visible status
	•	logs
	•	restart
	•	identity configuration

For MCP/VS Code integrations

Add:
	•	guided setup
	•	test connection
	•	show registered agent in UI after setup

The point is:
the app should bootstrap usage, not force users to manually assemble it from prose.

⸻

GUI transition plan

Now for the GUI question in the context of this bigger upgrade:

Recommendation

Build the GUI as the new primary operator surface, but keep the Go backend and APIs as source of truth.

Best default path
	•	Go backend stays
	•	desktop GUI becomes the primary app
	•	current web UI becomes fallback/admin/debug surface until parity

Why this is the right time

Because the same work needed for the GUI also fixes current product friction:
	•	typed controls
	•	better onboarding
	•	proper task inspectors
	•	run visibility
	•	agent health surfaces

⸻

GUI priorities

1. Dashboard

Needs to answer:
	•	what is running
	•	what is blocked
	•	what failed
	•	which agents are alive
	•	what needs approval

2. Tasks

Need:
	•	board view
	•	list view
	•	inspector
	•	dependency-aware creation
	•	templates
	•	approval flow

3. Runs

Need:
	•	recent runs list
	•	live run detail
	•	logs
	•	steps
	•	artifacts
	•	related task
	•	related agent

4. Agents

Need:
	•	registered agents
	•	online/offline
	•	current task
	•	last heartbeat
	•	capabilities
	•	recent failures
	•	stale alerts

5. Onboarding

Need:
	•	create/open project
	•	connect agent
	•	run built-in worker
	•	validate repo path
	•	test API health

⸻

Typed inputs: yes, this is mandatory

You’re right about selectors.

Any field with a known domain should not be raw text.

Replace text inputs with typed controls for:
	•	project ID → dropdown
	•	parent task → searchable picker
	•	dependency tasks → multi-select picker
	•	task type → dropdown
	•	priority → segmented or slider/stepper
	•	status → dropdown
	•	approval state → badges/actions
	•	agent → dropdown
	•	run → selectable list
	•	task ID for context packs → task picker, not text
	•	limits → bounded numeric fields
	•	booleans → toggles

Also add:
	•	autocomplete for tags
	•	templates for recurring task types
	•	file pickers for repo-linked actions
	•	safe defaults everywhere

This will cut operator error massively.

⸻

What else I think you should change

1. Stop treating the web pages as separate islands

Right now too many pages are thin placeholders built around one-off fetch logic.

You need a shared UI/service contract for:
	•	pagination
	•	loading
	•	empty states
	•	errors
	•	filters
	•	sorting
	•	detail loading

Even before a full GUI, this reduces breakage.

⸻

2. Add a “system health audit” page

Because the repo is public now, you need a single page that checks:
	•	DB migrations applied
	•	default project exists
	•	agent registration working
	•	claim-next working
	•	runs endpoint working
	•	events stream working
	•	context pack create/detail working
	•	repo indexing state
	•	policies loaded
	•	workdir valid

This is huge for trust.

⸻

3. Add a “demo mode” / sample data mode

Public repos need a great first 5 minutes.

Add one command or one button that:
	•	seeds sample project
	•	seeds sample tasks
	•	creates sample run
	•	creates sample events
	•	creates sample context pack
	•	shows agent lifecycle

Then the app looks alive immediately.

⸻

4. Add contract tests for every UI-backed endpoint

You already have many tests. Good.

Now add explicit “public experience” tests:
	•	agents page contract
	•	events page contract
	•	runs page contract
	•	context pack create/detail contract
	•	initial instructions endpoint correctness
	•	worker compatibility with current claim-next response

This should become release-blocking.

⸻

5. Version the onboarding flow

Treat onboarding as a first-class contract.

Have:
	•	onboarding v1
	•	agent bootstrap schema v1
	•	worker protocol v1

Then docs, UI, and worker stay aligned.

⸻

Suggested execution order

Wave 1 — trust repair
	1.	fix events page route
	2.	fix instructions endpoint examples
	3.	fix built-in worker contract mismatch
	4.	fix context pack input/detail mismatches
	5.	add recent runs list
	6.	audit all nav pages

Wave 2 — agent visibility
	7.	auto-upsert agent on claim
	8.	heartbeat integration for agents and leases
	9.	show real agent health in UI
	10.	show current run/task per agent

Wave 3 — onboarding redesign
	11.	add Connect Agent flow
	12.	add bootstrap generator
	13.	add first-run wizard
	14.	add demo mode

Wave 4 — GUI transition
	15.	desktop shell
	16.	dashboard
	17.	tasks + inspector
	18.	runs console
	19.	agents console
	20.	settings/health/audit

⸻

Default product decision

If you want the blunt recommendation:

Do not market this as “a web-based task queue server” anymore as the primary idea.
That undersells it and also encourages the wrong usage pattern.

Position it as:

a local agent orchestration console with tasking, runs, events, context, and operator control

That framing supports:
	•	GUI
	•	onboarding
	•	typed forms
	•	agent lifecycle
	•	observability
	•	public credibility

⸻

What matters most right now

The most important thing is not the GUI by itself.

The most important thing is:

make the canonical user flow exercise the actual platform.

Because right now a user can “successfully use” Cocopilot while skipping:
	•	agent registration
	•	heartbeats
	•	visible runs
	•	context packs
	•	observability

That means the product is succeeding in a way that makes it look broken.

That has to be reversed.

If you want, the next move should be turning this into a repo-grade upgrade plan with 15–25 concrete implementation tasks in priority order, written so you can drop them straight into Cocopilot.
Agreed. At this point this is release hardening + productization, not “fix a few bugs.”

I checked the supplied repo archive and the current public surface from source. I could not run the full Go test/build here because module download is blocked in this environment, so this is a static source audit, not a runtime certification. But there are enough verified issues to define a real upgrade program.

Verified blockers from the current source

1) The canonical onboarding path is stale and teaches the wrong product

In server/config.go:
	•	getInstructions() still tells the agent to use a curl loop as the main path
	•	it references the wrong claim URL once:
/api/v2/projects/default/tasks/claim-next
while the real default project is proj_default
	•	it tells the agent to “always use curl,” which bypasses the richer platform behavior

That means your first-run flow is actively steering users away from full platform usage.

⸻

2) The built-in worker is out of contract with the current API

In internal/worker/worker.go:
	•	workerClaimNext() expects either:
	•	result["envelope"]
	•	or result["task_id"]
	•	but current claim handlers return:
	•	task
	•	lease
	•	run
	•	context

So the reference worker is stale against the current claim-next contract.

Also, the worker:
	•	does not register the agent first
	•	does not send agent heartbeats
	•	does not heartbeat leases
	•	does not create run steps/logs/artifacts
	•	completes with a minimal payload only

That makes the “official worker” a weak demo of the system.

⸻

3) Agent visibility is structurally broken for the main usage path

The claim handlers require agent_id, but claiming a task does not auto-create or update an agent record.

Relevant files:
	•	server/handlers_v2_tasks.go
	•	server/handlers_v2_agents.go
	•	internal/dbstore/agents.go

Result:
	•	an agent can claim tasks
	•	leases/runs can exist
	•	but /api/v2/agents can still show nothing useful

So the UI is not wrong to look empty — the lifecycle is incomplete.

⸻

4) The Events page is broken by a route mismatch

This is a confirmed bug.

In server/routes.go:
	•	nav route is /events-browser

In server/ui_management.go:
	•	eventsBrowserHandler() only accepts r.URL.Path == "/events"

So the nav points to a page the handler rejects.

That is a public-facing failure.

⸻

5) Multiple public nav pages are still placeholder-grade

Verified in server/ui_pages.go and server/routes.go.

Pages still built as placeholders or partial tools:
	•	Agents
	•	Runs
	•	Memory
	•	Audit
	•	Repo
	•	Context Packs
	•	Task Graphs

This is the wrong state for a public repo if those pages are in the main navigation.

⸻

6) Context Packs UI has hard contract mismatches

In server/ui_pages.go and server/handlers_v2_projects.go:
	•	create page suggests task ID like task_abc123
	•	API requires task_id to be a positive integer
	•	detail page reads contents.files
	•	create handler writes file metadata under contents.repo_files.files

So even valid context packs can render as “no files.”

⸻

7) Runs UX is not public-grade

The top-level Runs page is basically just:
	•	type a run ID manually
	•	open /runs/{id}

There is no:
	•	recent runs list
	•	current runs list
	•	filtering
	•	discoverability

That is not acceptable for a public orchestration product.

⸻

8) The UI still forces raw text where the system already knows the type

Verified across server/ui_pages.go and server/ui_management.go.

Examples:
	•	project IDs
	•	task IDs
	•	dependency IDs
	•	event types
	•	memory scope/key
	•	context pack task ID
	•	workdir path
	•	parent/dependency relationships

This causes operator error and makes the product feel lower-end than the backend actually is.

⸻

9) The supplied archive is not release-clean

The uploaded zip contains:
	•	.git/
	•	__MACOSX/
	•	tasks.db
	•	tasks.db-shm
	•	tasks.db-wal

Your repo has guards for this:
	•	.gitignore
	•	scripts/package.sh
	•	CI artifact guard

So one of two things is true:
	•	the uploaded archive was not produced by the release path
	•	or the release discipline is not actually being followed

Either way, public-facing packaging is not locked down.

⸻

10) Public docs and product positioning are behind the product you want

Current README still frames it as:

“A web-based task queue server…”

That undersells the product and supports the wrong user behavior.

If the real vision is:
	•	operator console
	•	agent orchestration
	•	runs/events/context/policies/memory
	•	local first control room

then the docs need to sell that, not “task queue server + curl loop.”

⸻

11) You have duplicate CI surfaces and no single public-surface release gate

There are overlapping workflows:
	•	.github/workflows/ci.yml
	•	.github/workflows/go-ci.yml

This is not fatal, but it is public-repo sloppiness unless intentional and clearly separated.

More importantly, there is no single release gate that says:
	•	all nav pages load
	•	first-run onboarding works
	•	worker lifecycle is visible in UI
	•	no placeholder routes in public nav
	•	packaging is clean

You need that.

⸻

What this means

The repo is not in “tip top public condition” yet, even though there is a lot of solid backend work inside it.

The gap is not just code quality.
It is product integrity:
	•	onboarding flow
	•	lifecycle completeness
	•	UI/API parity
	•	page quality
	•	release hygiene
	•	positioning

⸻

What I would do now

Phase 0 — set the release bar

Define “public-ready” as these hard gates:
	1.	fresh clone → first visible working agent in under 5 minutes
	2.	every main nav page loads successfully
	3.	no page in primary nav is a placeholder
	4.	worker registers, heartbeats, claims, logs, completes, and shows in UI
	5.	no raw typed IDs where a picker can be used
	6.	release artifacts contain no DBs, .git, .DS_Store, __MACOSX, zips, or local junk
	7.	docs match the real onboarding path
	8.	all public-facing API/UI contracts are tested

Until those are true, do not call it polished.

⸻

Phase 1 — ship blockers

Fix these first:
	•	fix /events-browser route handling
	•	update getInstructions() and getDetailedInstructions() to the real protocol
	•	fix the built-in worker to parse current claim-next response
	•	add agent auto-upsert on claim or mandatory registration in bootstrap flow
	•	fix context pack create/detail mismatches
	•	upgrade runs page from manual lookup to actual list/discovery
	•	remove or hide placeholder-grade pages from primary nav until upgraded

⸻

Phase 2 — product flow rewrite

Make the canonical flow:
	1.	launch Cocopilot
	2.	choose/open project
	3.	connect agent
	4.	agent registers
	5.	agent heartbeats
	6.	agent claims tasks
	7.	run/lease/events/logs appear live
	8.	task completes visibly

That flow should be owned by the product, not by prose pasted into another chat.

⸻

Phase 3 — typed UI everywhere

Replace raw text with selectors for:
	•	project
	•	task
	•	parent task
	•	dependencies
	•	task type
	•	priority
	•	status
	•	scope
	•	policy target
	•	agent
	•	run

This is mandatory.

⸻

Phase 4 — GUI transition

Do the GUI shift after Phase 1/2 are stable, not before.

Best direction:
	•	keep Go backend as source of truth
	•	desktop GUI becomes the primary operator interface
	•	current web UI becomes fallback/admin/debug until parity

The GUI should center on:
	•	dashboard
	•	tasks + inspector
	•	runs console
	•	agents console
	•	onboarding/connect-agent flow

⸻

Phase 5 — release engineering

Add one release-grade smoke suite that validates:
	•	root loads
	•	projects loads
	•	agents loads
	•	runs loads
	•	memory loads
	•	policies loads
	•	dependencies loads
	•	context packs loads
	•	events loads
	•	repo loads
	•	audit loads
	•	settings loads
	•	health loads

And one first-run e2e:
	•	create project
	•	register agent
	•	create task
	•	claim task
	•	heartbeat
	•	create run log/step/artifact
	•	complete task
	•	verify UI/API state

That should be non-negotiable.

⸻

My direct recommendation

Do not treat this as “a few fixes before GUI.”

Treat it as a full vNext public hardening program with three linked goals:
	1.	stabilize the public repo
	2.	make the full platform actually utilized
	3.	shift to a real operator GUI

That is the right framing.

The next move should be to turn this into a strict release-hardening backlog with:
	•	ship blockers
	•	P1/P2/P3 priorities
	•	acceptance criteria
	•	exact file targets

I can write that backlog next as 20–30 concrete implementation tasks ready to drop straight into Cocopilot.