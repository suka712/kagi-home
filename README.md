# hp

`hp` is a local-first, AI-assisted context and task organizer for long-horizon
goals. It stores everything as plain markdown files in a git repo (so it
syncs across machines via any git host), and uses a local [Ollama](https://ollama.com)
model to help you break goals into subtasks, extract action items from
documents, and reprioritize — always showing you a preview and asking for
confirmation before it touches anything.

## Prerequisites

- Go 1.25+
- Git
- [Ollama](https://ollama.com) running locally with a model pulled, e.g.:
  ```
  ollama pull gemma4:e2b
  ollama serve
  ```

## Build / install

```
go build -o hp ./cmd/hp
mv hp /usr/local/bin/   # or anywhere on your $PATH
```

## Quick start

```
hp init                                  # creates the vault at ~/.hp
hp context add --title Bio - <<'EOF'
Second-year SWE student from Vietnam, took a gap year, RA role in Taiwan...
EOF
hp goal add "Impress the professor for the master's program" --target 2027-02-01
hp goal add "Land a remote SWE job paying \$900+/month"

hp ai plan --goal <goal-id>              # AI proposes subtasks, you approve each
hp task list --tree
hp ai prioritize                         # AI proposes priority changes, you approve each
hp ai extract --from notes.txt --goal <goal-id>   # pull action items out of a document
hp ai ask --goal <goal-id>                # AI asks what it needs to know; answers saved to context

hp sync                                   # commit + pull --rebase + push, if a remote is set
```

Task/goal IDs shown in listings are short (the last 8 characters of the
underlying ID) and can be used anywhere a `<id>` is expected — you don't need
to type the whole thing, just enough to be unambiguous.

## Where things live

Two separate directories, on purpose:

- **This repo** — the Go source for the `hp` binary. No personal data.
- **The vault** — your goals, tasks, and background context. Its own git
  repo, created by `hp init`, that you push to a **private** remote of your
  choice (GitHub, GitLab, a self-hosted git server — anything works, since
  `hp` just shells out to your normal `git`). Resolved in this order:
  `--dir <path>` flag > `$HP_HOME` env var > `~/.hp`.

```
$HP_HOME/
  config.yaml          # ollama host/model/temperature
  context/              background docs fed to the AI (bio, CV notes, constraints)
    qa-log.md            Q&A saved by `hp ai ask`
    files/                original uploaded files kept for reference (not parsed)
  goals/                 one markdown file per goal
  tasks/                 one markdown file per task (subtasks link via parent_id)
```

Every task/goal file is YAML frontmatter + a markdown body, so you can also
just open and hand-edit them directly — `hp` doesn't mind.

Every mutating command commits locally to the vault's git repo automatically,
so you get a full undo-able history for free (`git log` / `git revert` inside
the vault). `hp sync` is the *only* command that talks to a remote
(pull --rebase, then push) — nothing else touches the network.

## Commands

```
hp init [--dir path] [--remote url]

hp context add <file.md|.txt|- > [--title t]
hp context list

hp goal add "<title>" [--target YYYY-MM-DD]
hp goal list [--status s]
hp goal show <id>

hp task add "<title>" [--goal id] [--parent id] [--priority 1-5] [--due date]
hp task list [--goal id] [--status s] [--priority n] [--tree]
hp task show <id>
hp task edit <id> [--priority n] [--status s] [--due date] [--title t]
hp task done <id>

hp ai plan --goal <id> [--yes]             propose subtasks for a goal
hp ai prioritize [--goal <id>] [--yes]     propose priority changes across open tasks
hp ai extract --from <file|-> [--goal <id>] [--task <id>] [--title t] [--yes]
hp ai ask --goal <id>                      AI asks clarifying questions interactively

hp sync
hp status          # overdue / due soon / open tasks by goal
hp                  # same as `hp status`
```

`--yes` on any `hp ai *` command accepts every proposal without prompting;
by default each proposed task/priority-change/action-item is shown to you
individually with `[y/N/q]`.

## Config

`$HP_HOME/config.yaml`:

```yaml
ollama:
  host: http://localhost:11434
  model: gemma4:e2b
  temperature: 0.3
```

## Status

The CLI is the primary interface today. An interactive TUI dashboard
(bubbletea, for browsing/prioritizing visually) is a planned follow-up, built
on top of the same `internal/store` and `internal/ai` packages.
