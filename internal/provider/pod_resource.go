package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure interface compliance
var _ resource.Resource = &PodResource{}
var _ resource.ResourceWithImportState = &PodResource{}

func NewPodResource() resource.Resource {
	return &PodResource{}
}

// PodResource defines the resource implementation
type PodResource struct {
	client *Client
}

// PodResourceModel describes the resource data model
type PodResourceModel struct {
	ID                types.String `tfsdk:"id"`
	Name              types.String `tfsdk:"name"`
	ImageName         types.String `tfsdk:"image_name"`
	GpuTypeID         types.String `tfsdk:"gpu_type_id"`
	GpuCount          types.Int64  `tfsdk:"gpu_count"`
	VolumeInGb        types.Int64  `tfsdk:"volume_in_gb"`
	ContainerDiskInGb types.Int64  `tfsdk:"container_disk_in_gb"`
	CloudType         types.String `tfsdk:"cloud_type"`
	Ports             types.String `tfsdk:"ports"`
	VolumeMountPath   types.String `tfsdk:"volume_mount_path"`
	DockerArgs        types.String `tfsdk:"docker_args"`
	Env               types.Map    `tfsdk:"env"`
	MinVcpuCount      types.Int64  `tfsdk:"min_vcpu_count"`
	MinMemoryInGb     types.Int64  `tfsdk:"min_memory_in_gb"`
	NetworkVolumeID   types.String `tfsdk:"network_volume_id"`
	TemplateID        types.String `tfsdk:"template_id"`
	DataCenterID      types.String `tfsdk:"data_center_id"`
	SupportPublicIP   types.Bool   `tfsdk:"support_public_ip"`
	StartSSH          types.Bool   `tfsdk:"start_ssh"`
	MachineID         types.String `tfsdk:"machine_id"`
	PodHostID         types.String `tfsdk:"pod_host_id"`
}

func (r *PodResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pod"
}

func (r *PodResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a RunPod GPU pod.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier of the pod.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the pod.",
				Required:    true,
			},
			"image_name": schema.StringAttribute{
				Description: "The Docker image to use for the pod.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"gpu_type_id": schema.StringAttribute{
				Description: "The ID of the GPU type to use (e.g., 'NVIDIA RTX A6000').",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"gpu_count": schema.Int64Attribute{
				Description: "The number of GPUs to allocate.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(1),
				Validators: []validator.Int64{
					int64validator.AtLeast(1),
				},
			},
			"volume_in_gb": schema.Int64Attribute{
				Description: "The size of the persistent volume in GB.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(0),
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
				Validators: []validator.Int64{
					int64validator.AtLeast(0),
				},
			},
			"container_disk_in_gb": schema.Int64Attribute{
				Description: "The size of the container disk in GB.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(20),
				Validators: []validator.Int64{
					int64validator.AtLeast(1),
				},
			},
			"cloud_type": schema.StringAttribute{
				Description: "The type of cloud to deploy on (ALL, SECURE, or COMMUNITY).",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("ALL"),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf("ALL", "SECURE", "COMMUNITY"),
				},
			},
			"ports": schema.StringAttribute{
				Description: "Ports to expose (e.g., '8888/http,22/tcp').",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"volume_mount_path": schema.StringAttribute{
				Description: "The path to mount the persistent volume.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("/workspace"),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"docker_args": schema.StringAttribute{
				Description: "Docker arguments to pass to the container.",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"env": schema.MapAttribute{
				Description: "Environment variables to set in the container.",
				Optional:    true,
				ElementType: types.StringType,
				PlanModifiers: []planmodifier.Map{
					// Env vars cannot be changed after pod creation
				},
			},
			"min_vcpu_count": schema.Int64Attribute{
				Description: "Minimum number of vCPUs required.",
				Optional:    true,
			},
			"min_memory_in_gb": schema.Int64Attribute{
				Description: "Minimum amount of memory in GB required.",
				Optional:    true,
			},
			"network_volume_id": schema.StringAttribute{
				Description: "The ID of a network volume to attach.",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"template_id": schema.StringAttribute{
				Description: "The ID of a template to use for the pod.",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"data_center_id": schema.StringAttribute{
				Description: "The ID of the data center to deploy in.",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"support_public_ip": schema.BoolAttribute{
				Description: "Whether to support a public IP address.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"start_ssh": schema.BoolAttribute{
				Description: "Whether to start SSH service.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"machine_id": schema.StringAttribute{
				Description: "The ID of the machine the pod is running on.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"pod_host_id": schema.StringAttribute{
				Description: "The host ID of the pod.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *PodResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *PodResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data PodResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating pod", map[string]interface{}{
		"name": data.Name.ValueString(),
	})

	// Build pod input
	input := &PodInput{
		Name:              data.Name.ValueString(),
		ImageName:         data.ImageName.ValueString(),
		GpuCount:          int(data.GpuCount.ValueInt64()),
		VolumeInGb:        int(data.VolumeInGb.ValueInt64()),
		ContainerDiskInGb: int(data.ContainerDiskInGb.ValueInt64()),
	}

	// Set GPU type
	input.GpuTypeID = data.GpuTypeID.ValueString()

	if !data.CloudType.IsNull() {
		input.CloudType = data.CloudType.ValueString()
	}
	if !data.Ports.IsNull() {
		input.Ports = data.Ports.ValueString()
	}
	if !data.VolumeMountPath.IsNull() {
		input.VolumeMountPath = data.VolumeMountPath.ValueString()
	}
	if !data.DockerArgs.IsNull() {
		input.DockerArgs = data.DockerArgs.ValueString()
	}
	if !data.Env.IsNull() {
		envMap := make(map[string]string)
		resp.Diagnostics.Append(data.Env.ElementsAs(ctx, &envMap, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		for k, v := range envMap {
			input.Env = append(input.Env, EnvVar{Key: k, Value: v})
		}
	}
	if !data.MinVcpuCount.IsNull() {
		input.MinVcpuCount = int(data.MinVcpuCount.ValueInt64())
	}
	if !data.MinMemoryInGb.IsNull() {
		input.MinMemoryInGb = int(data.MinMemoryInGb.ValueInt64())
	}
	if !data.NetworkVolumeID.IsNull() {
		input.NetworkVolumeID = data.NetworkVolumeID.ValueString()
	}
	if !data.TemplateID.IsNull() {
		input.TemplateID = data.TemplateID.ValueString()
	}
	if !data.DataCenterID.IsNull() {
		input.DataCenterID = data.DataCenterID.ValueString()
	}
	if !data.SupportPublicIP.IsNull() {
		input.SupportPublicIP = data.SupportPublicIP.ValueBool()
	}
	if !data.StartSSH.IsNull() {
		input.StartSSH = data.StartSSH.ValueBool()
	}

	// Create pod
	pod, err := r.client.CreatePod(input)
	if err != nil {
		resp.Diagnostics.AddError("Client Error",
			fmt.Sprintf("Unable to create pod: %s", err))
		return
	}

	// Update state from API response
	data.ID = types.StringValue(pod.ID)
	if pod.MachineID != "" {
		data.MachineID = types.StringValue(pod.MachineID)
	}
	if pod.Machine != nil && pod.Machine.PodHostID != "" {
		data.PodHostID = types.StringValue(pod.Machine.PodHostID)
	}

	tflog.Trace(ctx, "Created pod", map[string]interface{}{"id": pod.ID})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *PodResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data PodResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Reading pod", map[string]interface{}{"id": data.ID.ValueString()})

	pod, err := r.client.GetPod(data.ID.ValueString())
	if err != nil {
		tflog.Error(ctx, "Error reading pod", map[string]interface{}{"id": data.ID.ValueString(), "error": err.Error()})
		// Handle deleted resources gracefully
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "Pod not found") {
			tflog.Warn(ctx, "Pod not found, removing from state", map[string]interface{}{"id": data.ID.ValueString()})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error",
			fmt.Sprintf("Unable to read pod: %s", err))
		return
	}

	// Update state from API response - only update fields that the API returns
	// Preserve existing state values for fields the API doesn't return
	data.Name = types.StringValue(pod.Name)
	data.ImageName = types.StringValue(pod.ImageName)
	if pod.Machine != nil && pod.Machine.GpuTypeID != "" {
		data.GpuTypeID = types.StringValue(pod.Machine.GpuTypeID)
	}
	// If API doesn't return GpuTypeID, preserve existing state value (don't overwrite)

	data.GpuCount = types.Int64Value(int64(pod.GpuCount))
	data.VolumeInGb = types.Int64Value(int64(pod.VolumeInGb))
	data.ContainerDiskInGb = types.Int64Value(int64(pod.ContainerDiskInGb))

	if pod.Ports != "" {
		data.Ports = types.StringValue(pod.Ports)
	}
	if pod.VolumeMountPath != "" {
		data.VolumeMountPath = types.StringValue(pod.VolumeMountPath)
	}
	if pod.DockerArgs != "" {
		data.DockerArgs = types.StringValue(pod.DockerArgs)
	}
	if pod.MachineID != "" {
		data.MachineID = types.StringValue(pod.MachineID)
	}
	if pod.Machine != nil && pod.Machine.PodHostID != "" {
		data.PodHostID = types.StringValue(pod.Machine.PodHostID)
	}

	// The following fields are not returned by the API, so preserve state values:
	// - CloudType: already preserved from state (loaded above)
	// - SupportPublicIP: already preserved from state (loaded above)
	// - StartSSH: already preserved from state (loaded above)
	// - Env: already preserved from state (loaded above)
	// - MinVcpuCount: already preserved from state (loaded above)
	// - MinMemoryInGb: already preserved from state (loaded above)
	// - NetworkVolumeID: already preserved from state (loaded above)
	// - TemplateID: already preserved from state (loaded above)
	// - DataCenterID: already preserved from state (loaded above)

	// Handle cloud_type - set default if not in state
	if data.CloudType.IsNull() || data.CloudType.IsUnknown() {
		data.CloudType = types.StringValue("ALL")
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *PodResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state PodResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Updating pod", map[string]interface{}{
		"id": state.ID.ValueString(),
	})

	// RunPod has limited update capabilities - most changes require recreation
	// For now, we just update the name if possible (though this may not be supported)
	// Most fields use RequiresReplace so Terraform will recreate the resource

	// Preserve computed fields
	plan.ID = state.ID
	plan.MachineID = state.MachineID
	plan.PodHostID = state.PodHostID

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *PodResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data PodResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Terminating pod", map[string]interface{}{
		"id": data.ID.ValueString(),
	})

	err := r.client.TerminatePod(data.ID.ValueString())
	if err != nil {
		// Ignore "not found" errors during delete
		if strings.Contains(err.Error(), "not found") {
			return
		}
		resp.Diagnostics.AddError("Client Error",
			fmt.Sprintf("Unable to terminate pod: %s", err))
		return
	}

	tflog.Trace(ctx, "Terminated pod", map[string]interface{}{
		"id": data.ID.ValueString(),
	})
}

func (r *PodResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
