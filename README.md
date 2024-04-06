# Vault Secrets Plugin for Nexus Reporitory

This is a secrets plugin which talks to [(Sonatype) Nexus Repository](https://www.sonatype.com/products/sonatype-nexus-repository) server and will dynamically create/revoke local user with predefined role(s). This backend can be mounted multiple times to provide access to multiple Nexus Repository servers, and work with both Pro and OSS versions.

Using this plugin, you can limit the accidental exposure window of Nexus Repository user's credentials; useful for continuous integration servers.

> [!IMPORTANT]
> This plugin is in development process, use at your own risk.
