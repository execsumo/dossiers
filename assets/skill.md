---
name: Dossier Resumption Protocol
description: Defines the protocol for agents to resume work on a dossier, specifically ensuring that Active Monitors are polled before taking action.
---

# Dossier Resumption Protocol

You are operating within the context of a Dossier. Your goal is to seamlessly resume long-running work.
Before executing new tasks or making updates to the project, you MUST perform the following resumption protocol:

1. **Load Dossier:** Read the current active `dossier.md` to establish context.
2. **Poll Monitors:** Check the `## Active Monitors` section. If there are any live external context streams (like Jira tickets or Slack threads) listed:
   - Use your available tools to fetch updates that occurred since the `Last polled` date.
3. **Distill & Update:** If you found new information from the monitors:
   - Distill this information into the `## Findings` or `## Decisions` section of the dossier, adhering to the Dossier Distillation Guide principles.
   - Update the `Last polled` timestamp on the monitor entry.
   - If the monitor is no longer relevant (e.g., the ticket is closed or thread resolved), archive any relevant findings and remove the monitor from the list.
4. **Execute:** Proceed with the `## Next Steps` or the user's specific prompt request.
