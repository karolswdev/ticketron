# Releasing Ticketron

This document outlines the process for creating a new release of the Ticketron CLI (`tix`).

## Release Process

Releases are managed automatically using [GoReleaser](https://goreleaser.com/) and GitHub Actions.

1.  **Ensure `main` branch is up-to-date and stable:** All tests should be passing, and the code should be in a state ready for release.
2.  **Determine the new version number:** Follow [Semantic Versioning](https://semver.org/) (e.g., `v0.1.0`, `v0.2.0`, `v1.0.0`).
3.  **Create and push a Git tag:** Use the chosen version number prefixed with `v`.

    ```bash
    # Example for version 0.2.0
    git tag v0.2.0
    git push origin v0.2.0
    ```

4.  **Monitor the GitHub Actions workflow:** Pushing the tag triggers the `Release Ticketron` workflow defined in `.github/workflows/release.yml`. This workflow will:
    *   Build binaries for Linux, macOS (Darwin), and Windows (amd64 and arm64).
    *   Create archives (`.tar.gz` and `.zip`) containing the binaries and documentation.
    *   Generate checksums.
    *   Create a draft GitHub Release associated with the tag, including the generated artifacts and a changelog based on commit messages.
5.  **Review and publish the draft release:** Navigate to the Releases section of the GitHub repository, review the draft release created by GoReleaser, make any necessary edits to the release notes, and then publish it.

The `version` information is automatically embedded into the built binaries using Go's `ldflags`.