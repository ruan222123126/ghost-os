package dev.ghostos.android.network

import org.junit.Assert.assertEquals
import org.junit.Test

class ArtifactFilenameResolverTest {
    private val resolver = ArtifactFilenameResolver()

    @Test
    fun `resolve uses content disposition filename when present`() {
        val filename = resolver.resolve(
            contentDisposition = "attachment; filename=\"report.md\"",
            fallbackArtifactId = "artifact-1",
            mimeType = "application/json",
        )

        assertEquals("report.md", filename)
    }

    @Test
    fun `resolve appends extension from mime type when needed`() {
        val filename = resolver.resolve(
            contentDisposition = null,
            fallbackArtifactId = "artifact-2",
            mimeType = "application/pdf",
        )

        assertEquals("artifact-2.pdf", filename)
    }

    @Test
    fun `resolve keeps artifact id when extension already exists`() {
        val filename = resolver.resolve(
            contentDisposition = null,
            fallbackArtifactId = "artifact-3.json",
            mimeType = "application/json",
        )

        assertEquals("artifact-3.json", filename)
    }
}
