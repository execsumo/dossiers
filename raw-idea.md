We're partnering on an app to solve a problem I'm having, and I'm sure others are as well. 

For context, I'm a (technically savvy) business user of claude code and other CLI coding agents. I use agents to work through many topics. Some are throw away, but many aren't. I might come back to the same topic in a few days or longer. I've tried using /resume sessions in claude and other tools, but that mixes up durable topics with throw-away, one-time sessions. Plus, ongoing sessions can become bloated and burn through too much context from exploration or false paths. Then there's the problem of being able to have various agents work on the same threads (claude, cursor, codex, antigravity, etc). I would like to be able to come back to a 'durable topic' and resume the agent session with the information required to pick off where we left, or to transition to another agent. 'handoff.md' is a great example of a similar pattern. that said, it's often not just captured context, but also attachments that I want / need to retain. A durable topic might have: written context on situation, decisions made by who, experiment results, links to references, queries, sources, powerpoints, emails, slack channels, slack threads, etc for reference and an accurate understanding, and cumulative progression.

unlike coding projects, the switching velocity between topics is much higher. I might touch on 20 'durable topics' (we need a better term) in a day. This means that treating each topic as a repo isn't suitable. It would be way too cumbersome to navigate topics. Further, I might have a session and realize it relates to a durable topic deep into the discussion and want to integrate with the existing topic, both to give the current session context, and to provide updated context for future sessions.

Some angles:
- as a user, I would like to be able to specify a session as a durable, ongoing topic to track, that can be resumed in future agent sessions.
- as a user, I would like to see all the durable threads that I have open without resolution that need to progress
- as a user, I would like to resume durable topics in new sessions with the agent having all the context we've discussed before.
- as a user, I would like the noise to be stripped out and prevented from being carried into future sessions resuming the durable topics. in other words, focus on critical information, without niceties and small talk. 
- as a user, I would like to be able to promote a session into a durable topic
- as a user, I would like to be able to connect ad hoc sessions to an existing thread in hindsight.
- as a user, I'd like to be able to cite decisions, the supporting logic, data, and slack conversations where we resolved a topic.
- as a user, I'd like to be able to share and have common threads with colleagues.
- I'd like a webapp UI, a terminal UI, and the ability for agents to recall threads within harnesses.

For the webapp UI, we'll also build in a 'llm wrapper' so that users can engage with the threads right in the web ui, with a preset llm / model. 