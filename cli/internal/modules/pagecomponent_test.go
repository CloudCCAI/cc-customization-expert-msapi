package modules

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"cloudcc-customization-expert-go/internal/jsonx"
)

func TestPageComponentCreateWritesCanonicalFiles(t *testing.T) {
	tmp := t.TempDir()
	var stderr bytes.Buffer
	if err := pageComponentCreate([]string{"cc-demo"}, &stderr, tmp); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(tmp, "frontend", "pagecomponents", "cc-demo")
	for _, file := range []string{
		filepath.Join(dir, "cc-demo.vue"),
		filepath.Join(dir, "components", "HelloWorld.vue"),
		filepath.Join(dir, "config.json"),
	} {
		if _, err := os.Stat(file); err != nil {
			t.Fatalf("expected generated file %s: %v", file, err)
		}
	}
	cfg, err := jsonx.ReadObjectFile(filepath.Join(dir, "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	if cfg["component"] != "component-cc-demo" {
		t.Fatalf("unexpected component: %#v", cfg["component"])
	}
}

func TestParsePageComponentVueData(t *testing.T) {
	content := `<template><div /></template>
<script>
export default {
  data() {
    return {
      componentInfo: {
        component: 'component-cc-demo',
        compName: "Demo",
        loadModel: "start"
      },
      propObj: { title: "hello" },
      events: {}
    };
  },
};
</script>`
	data, err := parsePageComponentVueData(content)
	if err != nil {
		t.Fatal(err)
	}
	info, _ := data["componentInfo"].(map[string]any)
	if info["component"] != "component-cc-demo" || info["loadModel"] != "start" {
		t.Fatalf("unexpected componentInfo: %#v", info)
	}
}

func TestCollectPageComponentDependencies(t *testing.T) {
	tmp := t.TempDir()
	root := filepath.Join(tmp, "frontend", "pagecomponents", "cc-demo")
	if err := os.MkdirAll(filepath.Join(root, "components"), 0755); err != nil {
		t.Fatal(err)
	}
	entry := filepath.Join(root, "cc-demo.vue")
	if err := os.WriteFile(entry, []byte(`<template><Child /></template>
<script>
import Child from "./components/Child.vue";
import helper from "./helper";
export default {};
</script>
<style>@import "./style.scss";</style>`), 0644); err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		filepath.Join(root, "components", "Child.vue"): "<template><div /></template>",
		filepath.Join(root, "helper.js"):               "export default {};",
		filepath.Join(root, "style.scss"):              ".demo { color: red; }",
	}
	for file, content := range files {
		if err := os.WriteFile(file, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	deps := collectPageComponentDependencies(entry, tmp)
	for _, key := range []string{
		"frontend/pagecomponents/cc-demo/cc-demo.vue",
		"frontend/pagecomponents/cc-demo/components/Child.vue",
		"frontend/pagecomponents/cc-demo/helper.js",
		"frontend/pagecomponents/cc-demo/style.scss",
	} {
		if _, ok := deps[key]; !ok {
			t.Fatalf("missing dependency %s in %#v", key, deps)
		}
	}
}

func TestPageComponentPublishRequiresPrebuiltBundleWithoutNode(t *testing.T) {
	tmp := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmp, "package.json"), []byte(`{"devConsoleConfig":{"accessToken":"token","pluginToken":"plugin","baseUrl":"https://example.invalid"}}`), 0644); err != nil {
		t.Fatal(err)
	}
	var createErr bytes.Buffer
	if err := pageComponentCreate([]string{"cc-demo"}, &createErr, tmp); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	err := pageComponentPublish([]string{"cc-demo"}, &stdout, &stderr, tmp)
	if err == nil {
		t.Fatal("expected missing prebuilt bundle error")
	}
	if !strings.Contains(err.Error(), "prebuilt UMD bundle") {
		t.Fatalf("expected prebuilt bundle error, got %v", err)
	}
	if !strings.Contains(err.Error(), "does not invoke Node/npm") {
		t.Fatalf("expected no Node/npm guidance, got %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(tmp, "frontend", "pagecomponents", "pagecomponentTemp.js")); !os.IsNotExist(statErr) {
		t.Fatalf("publish should not create temporary Node build entry, statErr=%v", statErr)
	}
}

func TestPageComponentPrebuiltBundlePathHonorsLocalConfig(t *testing.T) {
	tmp := t.TempDir()
	var stderr bytes.Buffer
	if err := pageComponentCreate([]string{"cc-demo"}, &stderr, tmp); err != nil {
		t.Fatal(err)
	}
	componentDir := filepath.Join(tmp, "frontend", "pagecomponents", "cc-demo")
	localConfig, err := jsonx.ReadObjectFile(filepath.Join(componentDir, "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	localConfig["bundlePath"] = "dist/custom.umd.min.js"
	bundle := filepath.Join(componentDir, "dist", "custom.umd.min.js")
	if err := os.MkdirAll(filepath.Dir(bundle), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(bundle, []byte("window.custom = true;"), 0644); err != nil {
		t.Fatal(err)
	}
	pub, err := readPageComponentPublishData(tmp, "cc-demo", localConfig, map[string]any{})
	if err != nil {
		t.Fatal(err)
	}
	resolved, _ := pageComponentPrebuiltBundlePath(tmp, "cc-demo", pub, localConfig, map[string]any{})
	if resolved != bundle {
		t.Fatalf("expected %s, got %s", bundle, resolved)
	}
}

func TestReadPageComponentPublishDataDoesNotCaptureRootConfig(t *testing.T) {
	tmp := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmp, "cloudcc-cli.config.json"), []byte(`{"use":"dev","dev":{"CloudCCDev":"secret"}}`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmp, ".env"), []byte(`TOKEN=secret`), 0644); err != nil {
		t.Fatal(err)
	}
	var stderr bytes.Buffer
	if err := pageComponentCreate([]string{"cc-demo"}, &stderr, tmp); err != nil {
		t.Fatal(err)
	}
	componentDir := filepath.Join(tmp, "frontend", "pagecomponents", "cc-demo")
	localConfig, err := jsonx.ReadObjectFile(filepath.Join(componentDir, "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	pub, err := readPageComponentPublishData(tmp, "cc-demo", localConfig, map[string]any{})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := pub.Dependencies["cloudcc-cli.config.json"]; ok {
		t.Fatalf("publish dependencies must not capture root CloudCC config: %#v", pub.Dependencies)
	}
	if _, ok := pub.Dependencies[".env"]; ok {
		t.Fatalf("publish dependencies must not capture environment files: %#v", pub.Dependencies)
	}
	if _, ok := pub.Dependencies["frontend/pagecomponents/cc-demo/cc-demo.vue"]; !ok {
		t.Fatalf("expected component source dependency, got %#v", pub.Dependencies)
	}
}

func TestPageComponentPackageDryRunReportsSafeFiles(t *testing.T) {
	tmp := t.TempDir()
	var stderr bytes.Buffer
	if err := pageComponentCreate([]string{"cc-demo"}, &stderr, tmp); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "cloudcc-cli.config.json"), []byte(`{"use":"dev","dev":{"CloudCCDev":"secret"}}`), 0644); err != nil {
		t.Fatal(err)
	}
	bundle := filepath.Join(tmp, "frontend", "build", "component-cc-demo.umd.min.js")
	if err := os.MkdirAll(filepath.Dir(bundle), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(bundle, []byte("window.CCDemo=true;"), 0644); err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	if err := pageComponentPackage([]string{"cc-demo", tmp, "--dry-run"}, &stdout, tmp); err != nil {
		t.Fatal(err)
	}
	out := stdout.String()
	if !strings.Contains(out, `"bundlePath"`) || !strings.Contains(out, `"frontend/pagecomponents/cc-demo/cc-demo.vue"`) {
		t.Fatalf("expected safe package preview, got %s", out)
	}
	var preview map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &preview); err != nil {
		t.Fatal(err)
	}
	for _, file := range preview["files"].([]any) {
		if strings.Contains(fmt.Sprint(file), "cloudcc-cli.config.json") {
			t.Fatalf("package preview files must not include root config, got %s", out)
		}
	}
	if strings.Contains(out, "CloudCCDev") {
		t.Fatalf("package preview must not include secret contents, got %s", out)
	}
}

func TestPluginResourceIsNotAlias(t *testing.T) {
	tmp := t.TempDir()
	var stdout, stderr bytes.Buffer
	err := Handle("create", "plugin", []string{"cc-demo"}, &stdout, &stderr, tmp)
	if err == nil {
		t.Fatal("plugin resource should not be accepted as a pagecomponent alias")
	}
	if _, statErr := os.Stat(filepath.Join(tmp, "plugins")); !os.IsNotExist(statErr) {
		t.Fatalf("plugin alias should not create files, statErr=%v", statErr)
	}
	if _, statErr := os.Stat(filepath.Join(tmp, "frontend", "pagecomponents")); !os.IsNotExist(statErr) {
		t.Fatalf("plugin alias should not create pagecomponent files, statErr=%v", statErr)
	}
}
