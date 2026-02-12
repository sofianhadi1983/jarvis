---
name: commit
description: Create well-formatted git commits. Use when the user wants to commit changes.
user_invocable: true
---

# Git Commit Skill

## Process
1. Run `git status` to see changed files
2. Run `git diff --staged` to review staged changes
3. If nothing staged, suggest which files to stage
4. Write a commit message following conventional commits format:
   - feat: new feature
   - fix: bug fix
   - docs: documentation
   - refactor: code restructuring
5. Run `git commit -m "message"`

## Rules
- Keep subject line under 72 characters
- Use imperative mood ("add feature" not "added feature")
- Don't commit .env or credential files
