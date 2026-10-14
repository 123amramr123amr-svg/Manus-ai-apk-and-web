# Keep the JS bridge reachable from the WebView.
-keepclassmembers class im.manus.webview.MainActivity$Bridge {
    public *;
}
-keepattributes JavascriptInterface
