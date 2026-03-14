package dev.ghostos.android.ui.theme

import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Typography
import androidx.compose.material3.lightColorScheme
import androidx.compose.runtime.Composable
import androidx.compose.ui.graphics.Color

private val GhostLightColorScheme = lightColorScheme(
    primary = Color(0xFF2A5FFF),
    onPrimary = Color(0xFFFFFFFF),
    primaryContainer = Color(0xFFDCE6FF),
    onPrimaryContainer = Color(0xFF11224A),
    secondary = Color(0xFF5B6472),
    onSecondary = Color(0xFFFFFFFF),
    tertiary = Color(0xFFB86E00),
    onTertiary = Color(0xFFFFFFFF),
    background = Color(0xFFF7F8FA),
    onBackground = Color(0xFF111827),
    surface = Color(0xFFFFFFFF),
    onSurface = Color(0xFF111827),
    surfaceVariant = Color(0xFFF1F4F9),
    onSurfaceVariant = Color(0xFF5B6472),
    outline = Color(0xFFD6DCE8),
    error = Color(0xFFCC3344),
    onError = Color(0xFFFFFFFF)
)

@Composable
fun GhostOSTheme(content: @Composable () -> Unit) {
    MaterialTheme(
        colorScheme = GhostLightColorScheme,
        typography = Typography(),
        content = content
    )
}
