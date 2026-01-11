# Walkthrough: Creating a Feature

Step-by-step guide to creating a new feature from idea to merge.

## Overview

| Step | Who | What |
|------|-----|------|
| 1 | Human | Describe the feature idea |
| 2 | AI | Create spec draft |
| 3 | Human | Review and refine spec |
| 4 | AI | Create implementation plan |
| 5 | Human | Approve plan |
| 6 | AI | Implement feature |
| 7 | Human | Review and merge |

## Step 1: Start the Conversation

Open Copilot Chat and use the feature prompt:

```
/new-feature I want to add a command that lists all active tasks
```

Or attach the prompt file and describe what you want.

## Step 2: AI Creates Spec

The AI will:
1. Allocate an ID (e.g., `FEAT-002`)
2. Create `work/specs/FEAT-002.md`
3. Fill in what it understands

You'll see a spec with:
- Summary
- User Story
- Acceptance Criteria (draft)

## Step 3: Review the Spec

Open `work/specs/FEAT-002.md` and check:

- [ ] Does the Summary capture your intent?
- [ ] Is the User Story accurate?
- [ ] Are the Acceptance Criteria complete?

**If changes needed:**
```
The acceptance criteria should also include error handling for invalid input
```

**If approved:**
```
Spec looks good, please create the implementation plan
```

## Step 4: AI Creates Plan

The AI creates `work/plans/FEAT-002-plan.md` with:
- High-level checklist
- Detailed steps with files to modify
- Dependencies and risks

## Step 5: Approve the Plan

Review the plan's Checklist section:

- [ ] Are all necessary steps included?
- [ ] Does the approach make sense?
- [ ] Are risks identified?

**If changes needed:**
```
Please add a step for updating the README with the new command
```

**If approved:**
```
Plan approved, please implement
```

## Step 6: AI Implements

The AI will:
1. Create branch `feat/FEAT-002-list-tasks`
2. Implement each step
3. Write tests
4. Commit with conventional messages
5. Update progress in the plan

You can watch progress in `work/plans/FEAT-002-plan.md`.

## Step 7: Review and Merge

When AI presents completed work:

1. **Check tests pass:**
   ```bash
   go test ./...
   ```

2. **Review changes:**
   Look at the commits or diff

3. **Check for deferred items:**
   See if anything was added to `BACKLOG.md`

4. **Merge:**
   ```bash
   git checkout main
   git merge feat/FEAT-002-list-tasks
   git push
   ```

5. **Clean up:**
   ```bash
   git branch -d feat/FEAT-002-list-tasks
   ```

## Tips

- **Be specific** in your initial description — saves revision cycles
- **Check BACKLOG.md** before starting — related items might exist
- **Small features** are easier to review than large ones
- **Ask questions** if anything is unclear — AI will answer and add to FAQ
