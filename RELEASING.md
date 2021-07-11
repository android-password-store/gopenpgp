## Releasing new versions

The steps for releasing a new version are straightforward:

1. Bump the version `build.gradle.kts`
2. Create a new tag with the name `aps/$version`. For release v1.1.0, this tag should be `aps/v1.1.0`
3. Push the commit bumping the version and the tag to upstream. GitHub Actions will then create build the new version and upload it to Sonatype Nexus.
4. Sign in to [Sonatype Nexus], go to `Staging Repositories`, select the automatically generated repository from the previous step, and click `Close`.
5. Once closing succeeds, click `Release` and ensure the 'Automatically drop this repository' option is checked in the following dialog.
6. That's it!

[Sonatype Nexus]: https://oss.sonatype.org/
