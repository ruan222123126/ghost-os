package dev.ghostos.android.network

private const val OCTET_STREAM_MIME = "application/octet-stream"

internal class ArtifactFilenameResolver {
    private val filenamePattern = Regex("""filename=\"([^\"]+)\"""")

    fun resolve(contentDisposition: String?, fallbackArtifactId: String, mimeType: String?): String {
        parseFilename(contentDisposition)?.let { return it }

        val normalizedMimeType = mimeType?.trim().orEmpty().ifBlank { OCTET_STREAM_MIME }
        val extension = extensionFromMimeType(normalizedMimeType)
        if (extension.isEmpty() || fallbackArtifactId.endsWith(extension)) {
            return fallbackArtifactId
        }
        return "$fallbackArtifactId$extension"
    }

    private fun parseFilename(contentDisposition: String?): String? {
        val header = contentDisposition?.trim().orEmpty()
        if (header.isEmpty()) {
            return null
        }
        return filenamePattern
            .find(header)
            ?.groupValues
            ?.getOrNull(1)
            ?.trim()
            ?.takeIf { it.isNotEmpty() }
    }

    private fun extensionFromMimeType(mimeType: String): String {
        return when (mimeType.substringBefore(';').trim().lowercase()) {
            "text/plain" -> ".txt"
            "text/markdown" -> ".md"
            "application/json" -> ".json"
            "application/pdf" -> ".pdf"
            "image/png" -> ".png"
            "image/jpeg" -> ".jpg"
            "image/gif" -> ".gif"
            else -> ""
        }
    }
}
