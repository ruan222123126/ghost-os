package dev.ghostos.android.model

data class DownloadedArtifact(
    val filename: String,
    val mimeType: String,
    val bytes: ByteArray,
)
