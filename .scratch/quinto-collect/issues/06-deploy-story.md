# 06 — Deploy story

**What to build:** Someone who is not you can stand up their own collector and point `quinto` at it, without reading the source.

**This ticket is why the collector was deferred in the first place.** The project's definition of done is a stranger installing the tool in under a minute. A bespoke collector means that stranger must also deploy a backend, create a database, configure a domain and manage a token before seeing a single number. This ticket exists to make that as close to painless as it can be — but it cannot make it as painless as pointing at a service that already exists.

Be honest in the documentation about what someone is signing up for: they own an endpoint now, and it is theirs to keep running.

**Blocked by:** 05.

**Status:** ready-for-agent

- [ ] Deploying a working collector is a single documented command
- [ ] Database creation and schema setup happen as part of that, not as manual steps
- [ ] Configuration — allowed origins, token — is documented and validated at deploy time with clear errors
- [ ] A first-time deployer gets from nothing to a recorded pageview in under ten minutes
- [ ] `quinto` is pointed at a self-deployed collector by configuration alone
- [ ] Privacy posture is documented: what is collected, what is not, why no consent banner is needed, and the retention story
- [ ] The README states plainly that self-deploying is the heavier path, and points at the hosted alternative for anyone who wants the fast one
