package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure interface compliance
var _ datasource.DataSource = &GpuTypesDataSource{}

func NewGpuTypesDataSource() datasource.DataSource {
	return &GpuTypesDataSource{}
}

// GpuTypesDataSource defines the data source implementation
type GpuTypesDataSource struct {
	client *Client
}

// GpuTypesDataSourceModel describes the data source data model
type GpuTypesDataSourceModel struct {
	ID       types.String       `tfsdk:"id"`
	GpuTypes []GpuTypeModel     `tfsdk:"gpu_types"`
	Filter   *GpuTypeFilterModel `tfsdk:"filter"`
}

type GpuTypeModel struct {
	ID             types.String `tfsdk:"id"`
	DisplayName    types.String `tfsdk:"display_name"`
	MemoryInGb     types.Int64  `tfsdk:"memory_in_gb"`
	SecureCloud    types.Bool   `tfsdk:"secure_cloud"`
	CommunityCloud types.Bool   `tfsdk:"community_cloud"`
}

type GpuTypeFilterModel struct {
	ID types.String `tfsdk:"id"`
}

func (d *GpuTypesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_gpu_types"
}

func (d *GpuTypesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches available GPU types from RunPod.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Identifier for this data source.",
				Computed:    true,
			},
			"gpu_types": schema.ListNestedAttribute{
				Description: "List of available GPU types.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "The ID of the GPU type (e.g., 'NVIDIA RTX A6000').",
							Computed:    true,
						},
						"display_name": schema.StringAttribute{
							Description: "The display name of the GPU type.",
							Computed:    true,
						},
						"memory_in_gb": schema.Int64Attribute{
							Description: "The amount of memory in GB.",
							Computed:    true,
						},
						"secure_cloud": schema.BoolAttribute{
							Description: "Whether this GPU type is available on secure cloud.",
							Computed:    true,
						},
						"community_cloud": schema.BoolAttribute{
							Description: "Whether this GPU type is available on community cloud.",
							Computed:    true,
						},
					},
				},
			},
		},
		Blocks: map[string]schema.Block{
			"filter": schema.SingleNestedBlock{
				Description: "Filter GPU types by ID.",
				Attributes: map[string]schema.Attribute{
					"id": schema.StringAttribute{
						Description: "Filter by GPU type ID (e.g., 'NVIDIA GeForce RTX 3090').",
						Optional:    true,
					},
				},
			},
		},
	}
}

func (d *GpuTypesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *Client, got: %T", req.ProviderData),
		)
		return
	}

	d.client = client
}

func (d *GpuTypesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data GpuTypesDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Reading GPU types")

	var gpuTypes []GpuType
	var err error

	// Check if we should filter by ID
	if data.Filter != nil && !data.Filter.ID.IsNull() {
		filterID := data.Filter.ID.ValueString()
		gpuType, err := d.client.GetGpuType(filterID)
		if err != nil {
			resp.Diagnostics.AddError("Client Error",
				fmt.Sprintf("Unable to read GPU type: %s", err))
			return
		}
		gpuTypes = []GpuType{*gpuType}
	} else {
		gpuTypes, err = d.client.ListGpuTypes()
		if err != nil {
			resp.Diagnostics.AddError("Client Error",
				fmt.Sprintf("Unable to list GPU types: %s", err))
			return
		}
	}

	// Convert to model
	data.GpuTypes = make([]GpuTypeModel, len(gpuTypes))
	for i, gt := range gpuTypes {
		data.GpuTypes[i] = GpuTypeModel{
			ID:             types.StringValue(gt.ID),
			DisplayName:    types.StringValue(gt.DisplayName),
			MemoryInGb:     types.Int64Value(int64(gt.MemoryInGb)),
			SecureCloud:    types.BoolValue(gt.SecureCloud),
			CommunityCloud: types.BoolValue(gt.CommunityCloud),
		}
	}

	// Set a placeholder ID
	data.ID = types.StringValue("gpu_types")

	tflog.Trace(ctx, "Read GPU types", map[string]interface{}{
		"count": len(gpuTypes),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
