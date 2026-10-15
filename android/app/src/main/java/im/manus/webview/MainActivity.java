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
import android.webkit.WebResourceResponse;
import android.webkit.WebSettings;
import android.webkit.WebView;
import android.webkit.WebViewClient;
import android.widget.EditText;
import android.widget.FrameLayout;
import android.widget.LinearLayout;
import android.widget.ProgressBar;
import android.widget.Toast;

import java.io.File;
import java.io.FileOutputStream;
import java.io.IOException;
import java.io.InputStream;
import java.nio.charset.StandardCharsets;
import java.util.HashMap;
import java.util.Map;

/**
 * Android shell with two modes:
 *
 * <ul>
 *   <li><b>Standalone agent</b> (default): the bundled agent app is served from
 *       the APK assets over {@code https://appassets.androidplatform.net/agent/},
 *       so the user only needs their own LLM API key — no server at all.</li>
 *   <li><b>Server mode</b>: the WebView loads a self-hosted Manus AI web server
 *       (the Go app in this repository) whose URL the user enters.</li>
 * </ul>
 */
public class MainActivity extends Activity {

    private static final String PREFS = "manus_prefs";
    private static final String KEY_URL = "server_url";
    private static final String KEY_MODE = "mode";
    private static final String MODE_AGENT = "agent";
    private static final String MODE_SERVER = "server";
    private static final String ASSET_ORIGIN = "https://appassets.androidplatform.net";
    private static final String AGENT_START = ASSET_ORIGIN + "/agent/index.html";
    private static final String OFFLINE_PAGE = "file:///android_asset/offline.html";
    private static final int FILE_CHOOSER_REQUEST = 1001;

    private WebView webView;
    private ProgressBar progressBar;
    private ValueCallback<Uri[]> filePathCallback;
    private String serverUrl = "";
    private String mode = MODE_AGENT;

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

        SharedPreferences preferences = getPreferences();
        mode = preferences.getString(KEY_MODE, MODE_AGENT);
        serverUrl = loadServerUrl();
        start();
    }

    /** Loads whichever frontend the current mode uses. */
    private void start() {
        if (MODE_SERVER.equals(mode)) {
            if (serverUrl.isEmpty()) {
                loadOfflinePage();
                showUrlDialog();
            } else {
                webView.loadUrl(serverUrl);
            }
            return;
        }
        webView.loadUrl(AGENT_START);
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
        // The standalone agent may talk to a local Ollama / LM Studio endpoint
        // over plain http on the user's own network, which needs mixed content.
        settings.setMixedContentMode(WebSettings.MIXED_CONTENT_ALWAYS_ALLOW);
        settings.setUserAgentString(settings.getUserAgentString() + " ManusAIAndroid/" + BuildConfig.VERSION_NAME);

        webView.setWebViewClient(new WebViewClient() {
            @Override
            public WebResourceResponse shouldInterceptRequest(WebView view, WebResourceRequest request) {
                return serveAsset(request.getUrl());
            }

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
                if (request.isForMainFrame() && MODE_SERVER.equals(mode)) {
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

    /**
     * Serves the bundled agent app from assets under a secure origin, so the
     * page can call external LLM APIs with the user's own key.
     */
    private WebResourceResponse serveAsset(Uri uri) {
        if (uri == null || !ASSET_ORIGIN.equals(uri.getScheme() + "://" + uri.getHost())) {
            return null;
        }
        String path = uri.getPath();
        if (path == null || path.isEmpty() || path.equals("/")) {
            path = "/agent/index.html";
        }
        while (path.startsWith("/")) {
            path = path.substring(1);
        }
        try {
            InputStream stream = getAssets().open(path);
            Map<String, String> headers = new HashMap<>();
            headers.put("Cache-Control", path.endsWith("index.html") ? "no-cache" : "max-age=604800");
            return new WebResourceResponse(mimeType(path), "utf-8", 200, "OK", headers, stream);
        } catch (IOException exception) {
            return null;
        }
    }

    private String mimeType(String path) {
        String lower = path.toLowerCase();
        if (lower.endsWith(".html")) return "text/html";
        if (lower.endsWith(".css")) return "text/css";
        if (lower.endsWith(".js")) return "text/javascript";
        if (lower.endsWith(".json") || lower.endsWith(".webmanifest")) return "application/json";
        if (lower.endsWith(".png")) return "image/png";
        if (lower.endsWith(".jpg") || lower.endsWith(".jpeg")) return "image/jpeg";
        if (lower.endsWith(".svg")) return "image/svg+xml";
        if (lower.endsWith(".ico")) return "image/x-icon";
        return "application/octet-stream";
    }

    /** Opens external links in the system browser and keeps the app on its origin. */
    private boolean handleUrl(String url) {
        if (url == null) {
            return false;
        }
        if (url.startsWith("file://") || url.startsWith("about:") || url.startsWith(ASSET_ORIGIN)) {
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
        mode = MODE_SERVER;
        getPreferences().edit().putString(KEY_URL, serverUrl).putString(KEY_MODE, mode).apply();
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

    /** Native dialog asking for the web server URL (server mode). */
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

    /** Lets the user switch between the standalone agent and a server. */
    private void showModeDialog() {
        String[] options = {getString(R.string.mode_agent), getString(R.string.mode_server)};
        new AlertDialog.Builder(this)
                .setTitle(R.string.mode_title)
                .setItems(options, (dialog, which) -> {
                    if (which == 0) {
                        mode = MODE_AGENT;
                        getPreferences().edit().putString(KEY_MODE, mode).apply();
                        start();
                    } else {
                        mode = MODE_SERVER;
                        getPreferences().edit().putString(KEY_MODE, mode).apply();
                        if (serverUrl.isEmpty()) {
                            showUrlDialog();
                        } else {
                            start();
                        }
                    }
                })
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

    /** JS bridge available to the web apps as window.Android. */
    private class Bridge {

        @JavascriptInterface
        public String getMode() {
            return mode;
        }

        @JavascriptInterface
        public String getServerUrl() {
            return serverUrl == null ? "" : serverUrl;
        }

        @JavascriptInterface
        public void setServerUrl(final String url) {
            runOnUiThread(() -> loadServer(url));
        }

        /** Opens the mode picker from inside the agent app. */
        @JavascriptInterface
        public void switchMode() {
            runOnUiThread(MainActivity.this::showModeDialog);
        }

        /** Opens the relevant settings dialog (URL for server mode, modes otherwise). */
        @JavascriptInterface
        public void openSettings() {
            runOnUiThread(() -> {
                if (MODE_SERVER.equals(mode)) {
                    showUrlDialog();
                } else {
                    showModeDialog();
                }
            });
        }

        @JavascriptInterface
        public void retry() {
            runOnUiThread(MainActivity.this::start);
        }

        /** Saves an exported chat/note into the app's documents folder. */
        @JavascriptInterface
        public void saveText(final String name, final String content) {
            try {
                File directory = new File(getExternalFilesDir(null), "manus");
                if (!directory.exists() && !directory.mkdirs()) {
                    throw new IOException("cannot create " + directory);
                }
                File target = new File(directory, name == null || name.isEmpty() ? "manus-export.md" : name);
                try (FileOutputStream output = new FileOutputStream(target)) {
                    output.write((content == null ? "" : content).getBytes(StandardCharsets.UTF_8));
                }
                final String path = target.getAbsolutePath();
                runOnUiThread(() -> Toast.makeText(MainActivity.this,
                        getString(R.string.saved_to) + " " + path, Toast.LENGTH_LONG).show());
            } catch (Exception exception) {
                final String message = exception.getMessage();
                runOnUiThread(() -> Toast.makeText(MainActivity.this,
                        getString(R.string.save_failed) + " " + message, Toast.LENGTH_LONG).show());
            }
        }

        @JavascriptInterface
        public void toast(final String message) {
            runOnUiThread(() -> Toast.makeText(MainActivity.this, message, Toast.LENGTH_SHORT).show());
        }
    }
}
