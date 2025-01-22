---
name: Roadmap planning
about: Use this template for tracking release roadmaps.
title: 'Roadmap for vXXXXX'
labels: roadmap
assignees: zimmski
---

# Highlights :star2:

-   TODO Define highlights of this release.

# Changes :bulb:

-   [ ] Development & Management 🛠️
    -   [ ] TODO Change with what and why as goal.
-   [ ] Documentation 📚
    -   [ ] TODO Change with what and why as goal.
-   [ ] Evaluation ⏱️
    -   [ ] TODO Change with what and why as goal.
-   [ ] Models 🤖
    -   [ ] TODO Change with what and why as goal.
-   [ ] Reports & Metrics 🗒️
    -   [ ] TODO Change with what and why as goal.
-   [ ] Operating Systems 🖥️
    -   [ ] TODO Change with what and why as goal.
-   [ ] Tools 🧰
    -   [ ] TODO Change with what and why as goal.
-   [ ] Tasks 🔢
    -   [ ] TODO Change with what and why as goal.
-   [ ] Closed PR / not-implemented issue 🚫
    -   [ ] TODO what and why with reason

# Details :mag:

-   [ ] TODO Take details from automatic release description of GitHub.

Release version of this roadmap issue:

> ❓ When should a release happen? Check the [`README`](../../README.md#when-and-how-to-release)!

-   [ ] Do a full evaluation with the version
    -   [ ] Exclude certain Openrouter models by default
        -   [ ] `nitro` cause they are just faster
        -   [ ] `extended` cause longer context windows don't matter for our tasks
        -   [ ] `free` and `auto` cause these are just "aliases" for existing models
    -   [ ] Exclude special-purpose models
        -   [ ] Vision models
        -   [ ] Roleplay and creative writing models
        -   [ ] Classification models
        -   [ ] Models with internet access (usually denoted by `-online` suffix)
        -   [ ] Models with extended context windows (usually denoted by `-1234K` suffix)
    -   [ ] Always prefer fine tuned (`-instruct`, `-chat`) models over a plain base model
-   [ ] Tag version (tag can be moved in case important merges happen afterwards)
-   [ ] For all issues of the current milestone, one by one, add them to the roadmap tasks (it is ok if a task has multiple issues) with the users that worked on it
    -   Fixed bugs should always be sorted into respective relevant categories and not in a generic "Bugs" category!
-   [ ] For all PRs of the current milestone, one by one, add them to the roadmap tasks (it is ok if a task has multiple issues) with the users that worked on it
    -   Fixed bugs should always be sorted into respective relevant categories and not in a generic "Bugs" category!
-   [ ] Search all issues for ...
    -   [ ] Unassigned issues that are closed, and assign them someone
    -   [ ] Issues without a milestone, and assign them a milestone
    -   [ ] Issues without a label, and assign them at least one label
-   [ ] Write the release notes:
    -   [ ] Use the tasks that are already there for the release note outline
    -   [ ] Add highlighted features based on the done tasks, sort by how many users would use the feature
-   [ ] Do the release for version X.Y.Z with a new major, minor or bugfix version
    -   [ ] Execute `go run scripts/eval-dev-quality-release/main.go X.Y.Z`
    -   [ ] Do release notes for version
    -   [ ] Set release as latest release
-   [ ] Prepare the next roadmap
    -   [ ] Create a milestone for the next release
    -   [ ] Create a new roadmap issue for the next release
        -   [ ] Move all open tasks/TODOs from this roadmap issue to the next roadmap issue.
        -   [ ] Move every comment of this roadmap issue as a TODO to the next roadmap issue. Mark when done with a :rocket: emoji.
-   [ ] Blog post containing evaluation results, new features and learnings
    -   [ ] Update README with blog post link and new header image
    -   [ ] Update repository link with blog post link
    -   [ ] https://github.com/symflower/eval-dev-quality/discussions
        -   [ ] Remove the previous announcements
        -   [ ] Add a "Deep dive: $blog-post-title" announcement for the blog post, unpin all others and pin this one
        -   [ ] Add a "v$version: $summary-of-highlights" announcement for the release, unpin all others and pin this one
    -   [ ] symflower.com
        -   [ ] Update "latest DevQualityEval deep dive" mentions
        -   [ ] Update DevQualityEval blog series lists with new entries
        -   [ ] Update LLM blog series lists with new entries
    -   [ ] Update payment process for supporting DevQualityEval
        -   [ ] New Stripe payment link for this version
        -   [ ] Update payment logic with new Google Drive folder of the evaluation
        -   [ ] Update payment link in this README
        -   [ ] Update payment link on symflower.com (except for the one deep dive that mentions exactly these results)
    -   [ ] Create an issue in the company tracker for Markus to announce the new deep dive on Twitter and LinkedIn
-   [ ] Close this issue
-   [ ] Close the current milestone
-   [ ] Announce release
-   [ ] Eat cake 🎂

TODO sort and sort out:

-   [ ] TODO
