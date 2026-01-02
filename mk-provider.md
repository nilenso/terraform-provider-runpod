# Building a Terraform Provider from Scratch

This guide covers creating a production-quality Terraform provider using the Terraform Plugin Framework, with comprehensive testing using terraform-plugin-testing.

## Project Structure

```
terraform-provider-{name}/
├── .github/
│   └── workflows/
│       ├── release.yml      # GoReleaser for publishing
│       └── test.yml         # CI: build, lint, acceptance tests
├── examples/
│   └── main.tf              # Example usage
├── internal/
│   └── provider/
│       ├── client.go                    # API client
│       ├── provider.go                  # Provider definition
│       ├── provider_test.go             # Provider tests & test helpers
│       ├── {resource}_resource.go       # Resource implementation
│       ├── {resource}_resource_test.go  # Resource acceptance tests
│       ├── {resource}_data_source.go    # Data source implementation
│       └── {resource}_data_source_test.go
├── .gitignore
├── .goreleaser.yml
├── .terraform-registry-manifest.json
├── go.mod
├── main.go
├── Makefile
└── README.md
```

## Step 1: Initialize the Project

```bash
mkdir terraform-provider-{name}
cd terraform-provider-{name}
go mod init github.com/{org}/terraform-provider-{name}
```

### go.mod Dependencies

```go
module github.com/{org}/terraform-provider-{name}

go 1.24.0

require (
    github.com/hashicorp/terraform-plugin-framework v1.17.0
    github.com/hashicorp/terraform-plugin-framework-validators v0.19.0
    github.com/hashicorp/terraform-plugin-go v0.29.0
    github.com/hashicorp/terraform-plugin-log v0.10.0
    github.com/hashicorp/terraform-plugin-testing v1.14.0
)
```

## Step 2: Main Entry Point

**main.go:**
```go
package main

import (
    "context"
    "flag"
    "log"

    "github.com/hashicorp/terraform-plugin-framework/providerserver"
    "github.com/{org}/terraform-provider-{name}/internal/provider"
)

var version = "dev"

func main() {
    var debug bool

    flag.BoolVar(&debug, "debug", false, "enable debug mode for debuggers like delve")
    flag.Parse()

    opts := providerserver.ServeOpts{
        Address: "registry.terraform.io/{org}/{name}",
        Debug:   debug,
    }

    err := providerserver.Serve(context.Background(), provider.New(version), opts)
    if err != nil {
        log.Fatal(err.Error())
    }
}
```

## Step 3: API Client

**internal/provider/client.go:**

The client should:
- Handle authentication
- Implement retry logic with exponential backoff for rate limiting
- Use a mutex for sequential API calls if required by the API
- Provide typed request/response structs
- Return meaningful errors

```go
package provider

import (
    "bytes"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "sync"
    "time"
)

const defaultBaseURL = "https://api.example.com/v1"

type Client struct {
    baseURL    string
    apiKey     string
    httpClient *http.Client
    mu         sync.Mutex // ensures sequential API calls if needed
}

func NewClient(apiKey string) *Client {
    return &Client{
        baseURL: defaultBaseURL,
        apiKey:  apiKey,
        httpClient: &http.Client{
            Timeout: 30 * time.Second,
        },
    }
}

func (c *Client) doRequest(method, endpoint string, body interface{}) ([]byte, error) {
    c.mu.Lock()
    defer c.mu.Unlock()

    url := c.baseURL + endpoint

    var jsonBody []byte
    var err error
    if body != nil {
        jsonBody, err = json.Marshal(body)
        if err != nil {
            return nil, fmt.Errorf("failed to marshal request body: %w", err)
        }
    }

    // Retry with exponential backoff for rate limiting
    maxRetries := 5
    baseDelay := 2 * time.Second

    for attempt := 0; attempt < maxRetries; attempt++ {
        var reqBody io.Reader
        if jsonBody != nil {
            reqBody = bytes.NewBuffer(jsonBody)
        }

        req, err := http.NewRequest(method, url, reqBody)
        if err != nil {
            return nil, fmt.Errorf("failed to create request: %w", err)
        }

        req.Header.Set("Content-Type", "application/json")
        req.Header.Set("Authorization", "Bearer "+c.apiKey)

        resp, err := c.httpClient.Do(req)
        if err != nil {
            return nil, fmt.Errorf("failed to execute request: %w", err)
        }

        respBody, err := io.ReadAll(resp.Body)
        resp.Body.Close()
        if err != nil {
            return nil, fmt.Errorf("failed to read response body: %w", err)
        }

        // Retry on 429 Too Many Requests or 503 Service Unavailable
        if resp.StatusCode == http.StatusTooManyRequests || 
           resp.StatusCode == http.StatusServiceUnavailable {
            if attempt < maxRetries-1 {
                delay := baseDelay * time.Duration(1<<attempt)
                time.Sleep(delay)
                continue
            }
        }

        if resp.StatusCode >= 400 {
            return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(respBody))
        }

        return respBody, nil
    }

    return nil, fmt.Errorf("max retries exceeded")
}

// Ping tests the API connection
func (c *Client) Ping() error {
    _, err := c.doRequest("GET", "/ping", nil)
    return err
}

// Define CRUD methods for each resource type
// func (c *Client) CreateResource(...) (*Resource, error)
// func (c *Client) GetResource(id string) (*Resource, error)
// func (c *Client) UpdateResource(id string, ...) (*Resource, error)
// func (c *Client) DeleteResource(id string) error
```

## Step 4: Provider Definition

**internal/provider/provider.go:**

```go
package provider

import (
    "context"
    "os"

    "github.com/hashicorp/terraform-plugin-framework/datasource"
    "github.com/hashicorp/terraform-plugin-framework/provider"
    "github.com/hashicorp/terraform-plugin-framework/provider/schema"
    "github.com/hashicorp/terraform-plugin-framework/resource"
    "github.com/hashicorp/terraform-plugin-framework/types"
)

var _ provider.Provider = &MyProvider{}

type MyProvider struct {
    version string
}

type MyProviderModel struct {
    APIKey types.String `tfsdk:"api_key"`
}

func New(version string) func() provider.Provider {
    return func() provider.Provider {
        return &MyProvider{version: version}
    }
}

func (p *MyProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
    resp.TypeName = "runpod"
    resp.Version = p.version
}

func (p *MyProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
    resp.Schema = schema.Schema{
        Description: "Interact with Example API.",
        Attributes: map[string]schema.Attribute{
            "api_key": schema.StringAttribute{
                Description: "API key. Can also be set via RUNPOD_API_KEY environment variable.",
                Optional:    true,
                Sensitive:   true,
            },
        },
    }
}

func (p *MyProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
    var config MyProviderModel

    resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
    if resp.Diagnostics.HasError() {
        return
    }

    // Get API key from config or environment
    apiKey := os.Getenv("RUNPOD_API_KEY")
    if !config.APIKey.IsNull() {
        apiKey = config.APIKey.ValueString()
    }

    if apiKey == "" {
        resp.Diagnostics.AddError(
            "Missing API Key",
            "Set the api_key value in configuration or use the RUNPOD_API_KEY environment variable.",
        )
        return
    }

    // Create and validate client
    client := NewClient(apiKey)
    if err := client.Ping(); err != nil {
        resp.Diagnostics.AddError(
            "Unable to Create API Client",
            "Error: "+err.Error(),
        )
        return
    }

    // Make client available to resources and data sources
    resp.DataSourceData = client
    resp.ResourceData = client
}

func (p *MyProvider) Resources(ctx context.Context) []func() resource.Resource {
    return []func() resource.Resource{
        NewMyResource,
    }
}

func (p *MyProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
    return []func() datasource.DataSource{
        NewMyDataSource,
    }
}
```

## Step 5: Resource Implementation

**internal/provider/{resource}_resource.go:**

Key concepts:
- Implement `resource.Resource` and `resource.ResourceWithImportState` interfaces
- Use typed model structs with `tfsdk` tags
- Handle plan modifiers for computed fields and replacement triggers
- Use validators for input validation
- Implement proper error handling with diagnostics

```go
package provider

import (
    "context"
    "fmt"
    "strings"

    "github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
    "github.com/hashicorp/terraform-plugin-framework/path"
    "github.com/hashicorp/terraform-plugin-framework/resource"
    "github.com/hashicorp/terraform-plugin-framework/resource/schema"
    "github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
    "github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
    "github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
    "github.com/hashicorp/terraform-plugin-framework/schema/validator"
    "github.com/hashicorp/terraform-plugin-framework/types"
    "github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure interface compliance
var _ resource.Resource = &MyResource{}
var _ resource.ResourceWithImportState = &MyResource{}

func NewMyResource() resource.Resource {
    return &MyResource{}
}

type MyResource struct {
    client *Client
}

type MyResourceModel struct {
    ID      types.String `tfsdk:"id"`
    Name    types.String `tfsdk:"name"`
    Type    types.String `tfsdk:"type"`
    // Add other fields
}

func (r *MyResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
    resp.TypeName = req.ProviderTypeName + "_my_resource"
}

func (r *MyResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
    resp.Schema = schema.Schema{
        Description: "Manages a resource.",
        Attributes: map[string]schema.Attribute{
            "id": schema.StringAttribute{
                Description: "The unique identifier.",
                Computed:    true,
                PlanModifiers: []planmodifier.String{
                    stringplanmodifier.UseStateForUnknown(),
                },
            },
            "name": schema.StringAttribute{
                Description: "The name of the resource.",
                Required:    true,
            },
            "type": schema.StringAttribute{
                Description: "The type of resource.",
                Required:    true,
                Validators: []validator.String{
                    stringvalidator.OneOf("type_a", "type_b", "type_c"),
                },
                PlanModifiers: []planmodifier.String{
                    stringplanmodifier.RequiresReplace(), // Changes force recreation
                },
            },
            "optional_field": schema.StringAttribute{
                Description: "An optional field with a default.",
                Optional:    true,
                Computed:    true,
                Default:     stringdefault.StaticString("default_value"),
            },
        },
    }
}

func (r *MyResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
    if req.ProviderData == nil {
        return
    }

    client, ok := req.ProviderData.(*Client)
    if !ok {
        resp.Diagnostics.AddError(
            "Unexpected Resource Configure Type",
            fmt.Sprintf("Expected *Client, got: %T", req.ProviderData),
        )
        return
    }

    r.client = client
}

func (r *MyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
    var data MyResourceModel

    resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
    if resp.Diagnostics.HasError() {
        return
    }

    tflog.Debug(ctx, "Creating resource", map[string]interface{}{
        "name": data.Name.ValueString(),
    })

    // Call API to create resource
    id, err := r.client.CreateResource(/* ... */)
    if err != nil {
        resp.Diagnostics.AddError("Client Error", 
            fmt.Sprintf("Unable to create resource: %s", err))
        return
    }

    data.ID = types.StringValue(id)

    tflog.Trace(ctx, "Created resource", map[string]interface{}{"id": id})

    resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *MyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
    var data MyResourceModel

    resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
    if resp.Diagnostics.HasError() {
        return
    }

    resource, err := r.client.GetResource(data.ID.ValueString())
    if err != nil {
        // Handle deleted resources gracefully
        if strings.Contains(err.Error(), "not found") {
            resp.State.RemoveResource(ctx)
            return
        }
        resp.Diagnostics.AddError("Client Error",
            fmt.Sprintf("Unable to read resource: %s", err))
        return
    }

    // Update state from API response
    data.Name = types.StringValue(resource.Name)
    // ... update other fields

    resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *MyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
    var data MyResourceModel

    resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
    if resp.Diagnostics.HasError() {
        return
    }

    tflog.Debug(ctx, "Updating resource", map[string]interface{}{
        "id": data.ID.ValueString(),
    })

    err := r.client.UpdateResource(data.ID.ValueString() /* ... */)
    if err != nil {
        resp.Diagnostics.AddError("Client Error",
            fmt.Sprintf("Unable to update resource: %s", err))
        return
    }

    resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *MyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
    var data MyResourceModel

    resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
    if resp.Diagnostics.HasError() {
        return
    }

    tflog.Debug(ctx, "Deleting resource", map[string]interface{}{
        "id": data.ID.ValueString(),
    })

    err := r.client.DeleteResource(data.ID.ValueString())
    if err != nil {
        resp.Diagnostics.AddError("Client Error",
            fmt.Sprintf("Unable to delete resource: %s", err))
        return
    }
}

func (r *MyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
    // Simple import by ID
    resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
    
    // Or complex import format: "parent_id/resource_id"
    // parts := strings.Split(req.ID, "/")
    // if len(parts) != 2 {
    //     resp.Diagnostics.AddError("Invalid Import ID",
    //         fmt.Sprintf("Expected format 'parent_id/id', got: %s", req.ID))
    //     return
    // }
    // resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("parent_id"), parts[0])...)
    // resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
}
```

## Step 6: Handling Different Field Types

### Write-Only Fields (CREATE-only, not returned by API)

These fields are sent during creation but the API never returns them. Without proper handling, Terraform will show spurious diffs on every plan.

```go
// In schema - use UseStateForUnknown to preserve the configured value
"gpu_type_ids": schema.ListAttribute{
    Description: "GPU types for the pod. Only used during creation.",
    ElementType: types.StringType,
    Optional:    true,
    PlanModifiers: []planmodifier.List{
        listplanmodifier.UseStateForUnknown(),
    },
},
```

In the Read function, simply don't touch these fields - the plan modifier preserves them.

### Immutable Fields (CREATE-only, but returned by API)

These fields can only be set during creation and changing them requires resource replacement:

```go
"network_volume_id": schema.StringAttribute{
    Description: "Network volume to attach. Cannot be changed after creation.",
    Optional:    true,
    PlanModifiers: []planmodifier.String{
        stringplanmodifier.RequiresReplace(),
    },
},
```

### Fields with Different Names in Request vs Response

Some APIs use different field names. Handle this in the client struct:

```go
// If API returns "image" but you want to use "image_name" in Terraform:
type Pod struct {
    // Use the response field name in the JSON tag
    ImageName string `json:"image,omitempty"` // or "imageName" - check actual API!
}
```

**Always verify actual API behavior** - OpenAPI specs can be outdated or incorrect.

## Step 7: Testing

Acceptance tests verify the provider works correctly against the real API. They follow a Create → Update → Import → Delete lifecycle.

### Test Setup

**internal/provider/provider_test.go:**

```go
package provider

import (
    "os"
    "testing"

    "github.com/hashicorp/terraform-plugin-framework/providerserver"
    "github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
    "runpod": providerserver.NewProtocol6WithError(New("test")()),
}

func testAccPreCheck(t *testing.T) {
    if os.Getenv("RUNPOD_API_KEY") == "" {
        t.Skip("RUNPOD_API_KEY must be set for acceptance tests")
    }
}
```

### Acceptance Tests

**internal/provider/pod_resource_test.go:**

```go
package provider

import (
    "fmt"
    "testing"

    "github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccPodResource_lifecycle(t *testing.T) {
    resource.Test(t, resource.TestCase{
        PreCheck:                 func() { testAccPreCheck(t) },
        ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
        Steps: []resource.TestStep{
            // Create
            {
                Config: testAccPodResourceConfig("test-pod", 20),
                Check: resource.ComposeAggregateTestCheckFunc(
                    resource.TestCheckResourceAttr("runpod_pod.test", "name", "test-pod"),
                    resource.TestCheckResourceAttr("runpod_pod.test", "volume_in_gb", "20"),
                    resource.TestCheckResourceAttrSet("runpod_pod.test", "id"),
                ),
            },
            // Update
            {
                Config: testAccPodResourceConfig("test-pod", 30),
                Check: resource.ComposeAggregateTestCheckFunc(
                    resource.TestCheckResourceAttr("runpod_pod.test", "volume_in_gb", "30"),
                ),
            },
            // Import
            {
                ResourceName:            "runpod_pod.test",
                ImportState:             true,
                ImportStateVerify:       true,
                ImportStateVerifyIgnore: []string{"gpu_type_ids"}, // write-only fields
            },
            // Delete happens automatically
        },
    })
}

func testAccPodResourceConfig(name string, volumeGb int) string {
    return fmt.Sprintf(`
resource "runpod_pod" "test" {
  name          = %[1]q
  image_name    = "runpod/pytorch:2.1.0-py3.10-cuda11.8.0-devel-ubuntu22.04"
  gpu_type_ids  = ["NVIDIA RTX A4000"]
  gpu_count     = 1
  volume_in_gb  = %[2]d
  container_disk_in_gb = 20
}
`, name, volumeGb)
}
```



## Step 8: Makefile

```makefile
.PHONY: build test testacc install clean fmt lint

BINARY=terraform-provider-{name}
VERSION?=0.1.0

UNAME_S := $(shell uname -s)
UNAME_M := $(shell uname -m)

ifeq ($(UNAME_S),Linux)
    OS=linux
endif
ifeq ($(UNAME_S),Darwin)
    OS=darwin
endif
ifeq ($(UNAME_M),x86_64)
    ARCH=amd64
endif
ifeq ($(UNAME_M),arm64)
    ARCH=arm64
endif

INSTALL_PATH=~/.terraform.d/plugins/registry.terraform.io/{org}/{name}/$(VERSION)/$(OS)_$(ARCH)

build:
	go build -o $(BINARY) -v

test:
	go test -v ./... -short

testacc:
	TF_ACC=1 go test -v ./internal/provider -timeout 30m -p 1 -parallel 1

testacc-one:
	TF_ACC=1 go test -v ./internal/provider -timeout 30m -run $(TEST)

install: build
	mkdir -p $(INSTALL_PATH)
	cp $(BINARY) $(INSTALL_PATH)/

clean:
	rm -f $(BINARY)

fmt:
	go fmt ./...

lint:
	golangci-lint run ./...
```

### Running Tests Locally

```bash
# Set your API key
export RUNPOD_API_KEY="your-api-key"

# Run all acceptance tests
make testacc

# Run a specific test
make testacc-one TEST=TestAccPodResource_lifecycle
```

## Step 9: CI/CD

**.github/workflows/test.yml:**

```yaml
name: Tests

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.24'

      - name: Build
        run: go build -v ./...

      - name: Run unit tests
        run: go test -v ./... -short

  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.24'

      - name: golangci-lint
        uses: golangci/golangci-lint-action@v6
        with:
          version: latest

  # Acceptance tests - only run on main after merge
  # PRs are validated by build+lint; acceptance tests run post-merge
  # For fork PRs, GitHub requires maintainer approval before any workflow runs
  acceptance:
    runs-on: ubuntu-latest
    if: github.event_name == 'push' && github.ref == 'refs/heads/main'
    needs: [build, lint]
    steps:
      - uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.24'

      - name: Set up Terraform
        uses: hashicorp/setup-terraform@v3
        with:
          terraform_wrapper: false

      - name: Run acceptance tests
        env:
          TF_ACC: "1"
          RUNPOD_API_KEY: ${{ secrets.RUNPOD_API_KEY }}
        run: go test -v ./internal/provider -timeout 30m -p 1 -parallel 1
        if: ${{ env.RUNPOD_API_KEY != '' }}
```

**Required GitHub Secrets:**
- `RUNPOD_API_KEY` - API key for acceptance tests
- `GPG_PRIVATE_KEY` - For signing releases
- `GPG_PASSPHRASE` - GPG key passphrase

**.github/workflows/release.yml:**

```yaml
name: Release

on:
  push:
    tags:
      - 'v*'

permissions:
  contents: write

jobs:
  goreleaser:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0
      - uses: actions/setup-go@v5
        with:
          go-version: '1.24'
      - uses: crazy-max/ghaction-import-gpg@v6
        id: import_gpg
        with:
          gpg_private_key: ${{ secrets.GPG_PRIVATE_KEY }}
          passphrase: ${{ secrets.GPG_PASSPHRASE }}
      - uses: goreleaser/goreleaser-action@v6
        with:
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          GPG_FINGERPRINT: ${{ steps.import_gpg.outputs.fingerprint }}
```

## Step 10: Release Configuration

**.goreleaser.yml:**

```yaml
version: 2

before:
  hooks:
    - go mod tidy

builds:
  - env:
      - CGO_ENABLED=0
    mod_timestamp: '{{ .CommitTimestamp }}'
    flags:
      - -trimpath
    ldflags:
      - '-s -w -X main.version={{.Version}}'
    goos:
      - linux
      - darwin
      - windows
    goarch:
      - amd64
      - arm64
    binary: '{{ .ProjectName }}_v{{ .Version }}'

archives:
  - format: zip
    name_template: '{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}'

checksum:
  name_template: '{{ .ProjectName }}_{{ .Version }}_SHA256SUMS'
  algorithm: sha256

signs:
  - artifacts: checksum
    args:
      - "--batch"
      - "--local-user"
      - "{{ .Env.GPG_FINGERPRINT }}"
      - "--output"
      - "${signature}"
      - "--detach-sign"
      - "${artifact}"

release:
  draft: false
  prerelease: auto
```

**.terraform-registry-manifest.json:**

```json
{
  "version": 1,
  "metadata": {
    "protocol_versions": ["6.0"]
  }
}
```

## Key Best Practices

### API Design

**Before writing any code, map out the API behavior:**

Create a table showing which fields are accepted/returned by each operation:

| Field | CREATE | UPDATE | READ | Notes |
|-------|--------|--------|------|-------|
| id | | | ✓ | Computed, server-generated |
| name | ✓ | ✓ | ✓ | Standard CRUD field |
| type | ✓ | | ✓ | Immutable after creation |
| gpu_type_ids | ✓ | | | Write-only, never returned |
| status | | | ✓ | Read-only, server-managed |

This analysis reveals:
- **Write-only fields** (CREATE only, not in READ): Use `UseStateForUnknown()` to preserve config value
- **Immutable fields** (CREATE only, returned in READ): Use `RequiresReplace()` plan modifier
- **Read-only fields** (READ only): Mark as `Computed: true`
- **Standard fields** (all operations): Normal Optional/Required handling

**Common API inconsistencies to watch for:**
- Field name differences between request and response (e.g., `imageName` vs `image`)
- Fields documented in OpenAPI spec but not actually returned by API
- Nested objects that may be null or have different structures

**Use PATCH for updates** when the API supports it (not PUT) - PATCH typically allows partial updates while PUT may require sending all fields.

### Handling Computed Fields and API Limitations

A common source of bugs is improper handling of computed fields, especially when the API doesn't return all fields. This causes spurious diffs on every `terraform plan`.

#### The Problem

There are three types of spurious diffs:

1. **`(known after apply)`** - Computed fields showing as unknown even when unchanged
2. **`+ field = "value"`** - Fields showing as additions on every plan  
3. **Perpetual diffs** - Optional+Computed fields with defaults that never stabilize

#### Understanding Terraform's Plan Sequence

```
┌─────────────────────────────────────────────────────────────────────────┐
│ 1. READ PHASE (happens first)                                           │
│    - Provider calls API to get current resource state                   │
│    - Provider populates state with actual values from API               │
└─────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│ 2. PLAN PHASE (happens after Read)                                      │
│    - Terraform compares: state (from Read) vs config (.tf files)        │
│    - Plan modifiers like UseStateForUnknown() are evaluated here        │
│    - Real changes show up because state already has new API values      │
└─────────────────────────────────────────────────────────────────────────┘
```

**Key insight**: `UseStateForUnknown()` does NOT hide real changes. It only prevents Terraform from marking already-known values as "unknown". Real changes are detected by comparing the previous state snapshot with the new state from Read.

#### Solution Pattern

For **computed-only fields** (e.g., `id`, `created_at`, `public_ip`):

```go
// Schema: Add UseStateForUnknown() to prevent "(known after apply)"
"public_ip": schema.StringAttribute{
    Computed: true,
    PlanModifiers: []planmodifier.String{
        stringplanmodifier.UseStateForUnknown(),
    },
},

// Read: Always set from API response
data.PublicIp = types.StringValue(apiResponse.PublicIp)
```

For **optional+computed fields with defaults** where API doesn't return the value:

```go
// Schema: Include both Default and UseStateForUnknown()
"cloud_type": schema.StringAttribute{
    Optional: true,
    Computed: true,
    Default:  stringdefault.StaticString("SECURE"),
    PlanModifiers: []planmodifier.String{
        stringplanmodifier.UseStateForUnknown(),
    },
},

// Read: Use API value if available, preserve state, or apply default
if apiResponse.CloudType != "" {
    data.CloudType = types.StringValue(apiResponse.CloudType)
} else if data.CloudType.IsNull() || data.CloudType.IsUnknown() {
    data.CloudType = types.StringValue("SECURE") // same as schema default
}
// If state has a value and API returns empty, state is preserved automatically
```

This three-tier fallback ensures:
1. API value wins if returned
2. Existing state value preserved if API doesn't return it
3. Default applied only for truly empty state (fixes corrupted state from buggy provider versions)

#### Required Imports for Plan Modifiers

```go
import (
    "github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
    "github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
    "github.com/hashicorp/terraform-plugin-framework/resource/schema/float64planmodifier"
    "github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
    "github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
    "github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
)
```

#### Checklist for Computed Fields

- [ ] Every computed-only field has `UseStateForUnknown()` plan modifier
- [ ] Every optional+computed field with a default has `UseStateForUnknown()` plan modifier
- [ ] Read function sets ALL computed fields from API response
- [ ] Read function handles missing API fields gracefully (preserve state or apply default)
- [ ] Default values in Read match schema defaults exactly
- [ ] Acceptance tests pass (they automatically verify no spurious diffs between steps)

### Resource Design
1. **Mark immutable fields** with `RequiresReplace()` plan modifier
2. **Use validators** for enum fields and format validation
3. **Provide sensible defaults** for optional fields
4. **Handle deleted resources** gracefully in Read (remove from state)
5. **Support import** for all resources

### Testing
1. **Test the full lifecycle**: Create → Read → Update → Delete
2. **Test import functionality** separately
3. **Test replacement triggers** for immutable fields
4. **Add delays between tests** to avoid rate limiting
5. **Use config helper functions** for DRY test configs
6. **Acceptance tests automatically verify no spurious diffs** - the test framework runs plan after each step and fails if unexpected changes are detected

### Error Handling
1. Use `resp.Diagnostics.AddError()` with clear, actionable messages
2. Include the underlying error in the message
3. Handle "not found" errors specially in Read (remove from state)

### Resources with Lifecycle States

Some resources have operational states (running/stopped). Handle these with dedicated API endpoints:

```go
func (r *MyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
    var plan, state MyResourceModel
    
    resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
    resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
    
    // Handle state transitions with dedicated endpoints
    if state.Status.ValueString() == "stopped" && plan.Status.ValueString() == "running" {
        if err := r.client.StartResource(plan.ID.ValueString()); err != nil {
            resp.Diagnostics.AddError("Client Error", 
                fmt.Sprintf("Unable to start resource: %s", err))
            return
        }
    }
    
    // Then handle other field updates with PATCH
    // ...
}
```

### Logging
1. Use `tflog.Debug()` for operation starts
2. Use `tflog.Trace()` for operation completions
3. Include relevant identifiers in log context
