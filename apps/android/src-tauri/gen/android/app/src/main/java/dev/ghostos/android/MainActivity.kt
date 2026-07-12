package dev.ghostos.android

import android.os.Bundle
import android.view.WindowManager

class MainActivity : TauriActivity() {
  @Suppress("DEPRECATION")
  override fun onCreate(savedInstanceState: Bundle?) {
    window.setSoftInputMode(WindowManager.LayoutParams.SOFT_INPUT_ADJUST_RESIZE)
    super.onCreate(savedInstanceState)
  }
}
