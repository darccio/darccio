# Flow

> anything which exists inside an organization in any time span. E.g. projects, conversations, tasks, etc.

In our dairy activity we are surrounded by conversations. Discussions. Debates. You name it.

A flow can be anything (based on its type) and it can last any amount of time (from forever to just minutes).

A flow has:

* **Original message:** message starting the flow. It has a subject (a.k.a. title).
* **Nexus:** one or more nexus where a flow belongs.
* **Messages:** discussion's messages. A message can only belong to one flow.
* **Type:** given a type, a flow acquires additional features: voting, edition, history, discussion turn, etc. A flow has only one type. A type can combine other types.
* **Parent Flow:** from which flow comes. It is not mandatory for standalone flows.
* **Related Flows:** flows that interact with another flow. Useful to track merged flows.

## Horizontality

Every member has equal rights over a flow based on nexus' settings.

They can reply to the original message. Any flow must allow replies.

In order to keep as much context and structure as possible, anyone can modify a flow (unless opposite is stated) with actions:

* **Fork:** from a given message, create a new flow. The new flow can have a new type. At source it is linked, in forked message, and at destiny, source is destiny flow's parent.
* **Merge:** create a new flow, from two other flows, with an unified collection of messages and an optional new original message.
* **Relate:** link it with another flow.
* **Mute:** for a given user, she won't receive notifications nor see it again.

This actions keep debate structured and avoid off-topics to take over main conversation. They also allow to manage duplication and information overload.

## Transparency, accessibility and privacy

Transparency and privacy come from nexus' settings where a flow belongs.

Flows must be accessible with permalink, as long as each message they contain.

## Cooperation

To simplify cooperation and discussion, common answers like agreement or rejection must be handled through specific mechanisms like +1/-1 buttons on messages.

Flows can be used to receive external feedback from different sources, like requests, criticism, press contacts, searches and mentions in social networks, leaks, ideas, etc.

A REST API is mandatory to allow third party integration. For example, [Huginn](https://github.com/cantino/huginn) integration for IFTTT-like features.
