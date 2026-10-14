package im.manus.webview;

import android.app.Activity;
import android.app.AlertDialog;
import android.content.Intent;
import android.content.SharedPreferences;
import android.net.Uri;
import android.os.Build;
import android.os.Bundle;
import android.text.InputType;
import android.view.View;
import android.view.ViewGroup;
import android.webkit.JavascriptInterface;
import android.webkit.ValueCallback;
import android.webkit.WebChromeClient;
import android.webkit.WebResourceError;
import android.webkit.WebResourceRequest;
import android.webkit.WebSettings;
import android.webkit.WebView;
import android.webkit.WebViewClient;
import android.widget.EditText;
import android.widget.FrameLayout;
import android.widget.LinearLayout;
import android.widget.ProgressBar;
import android.widget.Toast;

/**
 * Thin Android shell around the Manus AI web application: a WebView that loads
 * the self-hosted server URL and exposes a tiny JS bridge (window.Android) for
 * settings, retries and toasts.
 */
public class MainActivity extends Activity {

    private static final String PREFS = "manus_prefs";
    private static final String KEY_URL = "server_url";
    private static final int FILE_CHOOSER_REQUEST = 1001;
    private static final String OFFLINE_PAGE = "file:///android_asset/offline.html";

    private WebView webView;
    private ProgressBar progressBar;
    private ValueCallback<Uri[]> filePathCallback;
    private String serverUrl = "";

    @Override
    protected void onCreate(Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);

        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.KITKAT) {
            WebView.setWebContentsDebuggingEnabled(BuildConfig.DEBUG);
        }

        FrameLayout root = new FrameLayout(this);
        webView = new WebView(this);
        root.addView(webView, new FrameLayout.LayoutParams(
                ViewGroup.LayoutParams.MATCH_PARENT, ViewGroup.LayoutParams.MATCH_PARENT));

        progressBar = new ProgressBar(this, null, android.R.attr.progressBarStyleHorizontal);
        progressBar.setMax(100);
        progressBar.setVisibility(View.GONE);
        root.addView(progressBar, new FrameLayout.LayoutParams(
                ViewGroup.LayoutParams.MATCH_PARENT, dp(4)));

        setContentView(root);

        configureWebView();

        serverUrl = loadServerUrl();
        if (serverUrl.isEmpty()) {
            loadOfflinePage();
            showUrlDialog();
        } else {
            webView.loadUrl(serverUrl);
        }
    }

    private int dp(int value) {
        float density = getResources().getDisplayMetrics().density;
        return Math.round(value * density);
    }

    @SuppressWarnings("SetJavaScriptEnabled")
    private void configureWebView() {
        WebSettings settings = webView.getSettings();
        settings.setJavaScriptEnabled(true);
        settings.setDomStorageEnabled(true);
        settings.setDatabaseEnabled(true);
        settings.setLoadWithOverviewMode(true);
        settings.setUseWideViewPort(true);
        settings.setSupportZoom(false);
        settings.setAllowFileAccess(false);
        settings.setAllowContentAccess(true);
        settings.setMixedContentMode(WebSettings.MIXED_CONTENT_COMPATIBILITY_MODE);
        settings.setUserAgentString(settings.getUserAgentString() + " ManusAIAndroid/" + BuildConfig.VERSION_NAME);

        webView.setWebViewClient(new WebViewClient() {
            @Override
            public boolean shouldOverrideUrlLoading(WebView view, WebResourceRequest request) {
                return handleUrl(request.getUrl().toString());
            }

            @Override
            public void onPageFinished(WebView view, String url) {
                progressBar.setVisibility(View.GONE);
            }

            @Override
            public void onReceivedError(WebView view, WebResourceRequest request, WebResourceError error) {
                if (request.isForMainFrame()) {
                    loadOfflinePage();
                }
            }
        });

        webView.setWebChromeClient(new WebChromeClient() {
            @Override
            public void onProgressChanged(WebView view, int newProgress) {
                progressBar.setVisibility(newProgress >= 100 ? View.GONE : View.VISIBLE);
                progressBar.setProgress(newProgress);
            }

            @Override
            public boolean onShowFileChooser(WebView view, ValueCallback<Uri[]> callback,
                                             FileChooserParams params) {
                if (filePathCallback != null) {
                    filePathCallback.onReceiveValue(null);
                }
                filePathCallback = callback;

                Intent intent = params.createIntent();
                intent.addCategory(Intent.CATEGORY_OPENABLE);
                try {
                    startActivityForResult(intent, FILE_CHOOSER_REQUEST);
                    return true;
                } catch (Exception exception) {
                    filePathCallback = null;
                    Toast.makeText(MainActivity.this, R.string.file_chooser_error, Toast.LENGTH_SHORT).show();
                    return false;
                }
            }
        });

        webView.addJavascriptInterface(new Bridge(), "Android");
    }

    /** Opens external links in the system browser and keeps the app on its server. */
    private boolean handleUrl(String url) {
        if (url == null) {
            return false;
        }
        if (url.startsWith("file://") || url.startsWith("about:")) {
            return false;
        }
        String host = Uri.parse(url).getHost();
        String serverHost = Uri.parse(serverUrl).getHost();
        if (host != null && host.equals(serverHost)) {
            return false;
        }
        try {
            startActivity(new Intent(Intent.ACTION_VIEW, Uri.parse(url)));
        } catch (Exception ignored) {
            return false;
        }
        return true;
    }

    private void loadOfflinePage() {
        webView.loadUrl(OFFLINE_PAGE);
    }

    private void loadServer(String url) {
        if (url == null || url.trim().isEmpty()) {
            return;
        }
        serverUrl = normalizeUrl(url);
        getPreferences().edit().putString(KEY_URL, serverUrl).apply();
        webView.loadUrl(serverUrl);
    }

    private String normalizeUrl(String url) {
        String trimmed = url.trim();
        if (!trimmed.startsWith("http://") && !trimmed.startsWith("https://")) {
            trimmed = "https://" + trimmed;
        }
        while (trimmed.endsWith("/")) {
            trimmed = trimmed.substring(0, trimmed.length() - 1);
        }
        return trimmed;
    }

    private String loadServerUrl() {
        String stored = getPreferences().getString(KEY_URL, "");
        if (stored == null || stored.isEmpty()) {
            return BuildConfig.DEFAULT_SERVER_URL;
        }
        return stored;
    }

    private SharedPreferences getPreferences() {
        return getSharedPreferences(PREFS, MODE_PRIVATE);
    }

    /** Native dialog asking for the web app URL. */
    private void showUrlDialog() {
        final EditText input = new EditText(this);
        input.setInputType(InputType.TYPE_TEXT_VARIATION_URI);
        input.setHint(R.string.server_url_hint);
        input.setText(serverUrl);
        input.setSingleLine(true);

        LinearLayout container = new LinearLayout(this);
        container.setOrientation(LinearLayout.VERTICAL);
        int padding = dp(20);
        container.setPadding(padding, dp(8), padding, 0);
        container.addView(input);

        new AlertDialog.Builder(this)
                .setTitle(R.string.server_url_title)
                .setMessage(R.string.server_url_message)
                .setView(container)
                .setPositiveButton(R.string.action_save, (dialog, which) -> loadServer(input.getText().toString()))
                .setNegativeButton(R.string.action_cancel, null)
                .show();
    }

    @Override
    protected void onActivityResult(int requestCode, int resultCode, Intent data) {
        if (requestCode == FILE_CHOOSER_REQUEST) {
            Uri[] results = null;
            if (resultCode == RESULT_OK && data != null) {
                if (data.getClipData() != null) {
                    int count = data.getClipData().getItemCount();
                    results = new Uri[count];
                    for (int index = 0; index < count; index++) {
                        results[index] = data.getClipData().getItemAt(index).getUri();
                    }
                } else if (data.getData() != null) {
                    results = new Uri[]{data.getData()};
                }
            }
            if (filePathCallback != null) {
                filePathCallback.onReceiveValue(results);
                filePathCallback = null;
            }
            return;
        }
        super.onActivityResult(requestCode, resultCode, data);
    }

    @Override
    public void onBackPressed() {
        if (webView.canGoBack()) {
            webView.goBack();
            return;
        }
        super.onBackPressed();
    }

    @Override
    protected void onDestroy() {
        if (webView != null) {
            webView.destroy();
        }
        super.onDestroy();
    }

    /** JS bridge available to the web app as window.Android. */
    private class Bridge {

        @JavascriptInterface
        public String getServerUrl() {
            return serverUrl == null ? "" : serverUrl;
        }

        @JavascriptInterface
        public void setServerUrl(final String url) {
            runOnUiThread(() -> loadServer(url));
        }

        @JavascriptInterface
        public void openSettings() {
            runOnUiThread(MainActivity.this::showUrlDialog);
        }

        @JavascriptInterface
        public void retry() {
            runOnUiThread(() -> {
                if (serverUrl == null || serverUrl.isEmpty()) {
                    showUrlDialog();
                } else {
                    loadServer(serverUrl);
                }
            });
        }

        @JavascriptInterface
        public void toast(final String message) {
            runOnUiThread(() -> Toast.makeText(MainActivity.this, message, Toast.LENGTH_SHORT).show());
        }
    }
}
