package provider

import (
	"context"

	"github.com/YakDriver/regexache"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure FastSSMProvider satisfies various provider interfaces.
var _ provider.Provider = &FastSSMProvider{}

// var _ provider.ProviderWithFunctions = &FastSSMProvider{}

// FastSSMProvider defines the provider implementation.
type FastSSMProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// FastSSMProviderModel describes the provider data model.
// TODO pending nested objects, most likely need structs
type FastSSMProviderModel struct {
	AccessKey                 types.String `tfsdk:"access_key"`
	AllowedAccountIds         types.Set    `tfsdk:"allowed_account_ids"`
	AssumeRole                types.List   `tfsdk:"assume_role"`                   // nested
	AssumeRoleWithWebIdentity types.List   `tfsdk:"assume_role_with_web_identity"` // nested
	CustomCABundle            types.String `tfsdk:"custom_ca_bundle"`
	DefaultTags               types.Map    `tfsdk:"default_tags"`
	Endpoints                 types.Set    `tfsdk:"endpoints"` // nested
	ForbiddenAccountsIds      types.Set    `tfsdk:"forbidden_account_ids"`
	HTTPProxy                 types.String `tfsdk:"http_proxy"`
	HTTPSProxy                types.String `tfsdk:"https_proxy"`
	Insecure                  types.Bool   `tfsdk:"insecure"`
	IgnoreTags                types.List   `tfsdk:"ignore_tags"`
	MaxRetries                types.Int32  `tfsdk:"max_retries"`
	NoProxy                   types.String `tfsdk:"no_proxy"`
	Profile                   types.String `tfsdk:"profile"`
	Region                    types.String `tfsdk:"region"`
	RetryMode                 types.String `tfsdk:"retry_mode"`
	S3UserPathStyle           types.Bool   `tfsdk:"s3_use_path_style"`
	// S3USEast1RegionalEndpoint      types.String `tfsdk:"s3_us_east_1_regional_endpoint"`
	SecretKey                      types.String `tfsdk:"secret_key"`
	SharedConfigFiles              types.List   `tfsdk:"shared_config_files"`
	SharedCredentialsFiles         types.List   `tfsdk:"shared_credentials_files"`
	SkipCredentialsValidation      types.Bool   `tfsdk:"skip_credentials_validation"`
	SkipMetadataAPICheck           types.Bool   `tfsdk:"skip_metadata_api_check"`
	SkipRegionValidation           types.Bool   `tfsdk:"skip_region_validation"`
	SkipRequestingAccountId        types.Bool   `tfsdk:"skip_requesting_account_id"`
	STSRegion                      types.String `tfsdk:"sts_region"`
	Token                          types.String `tfsdk:"token"`
	TokenBucketRateLimiterCapacity types.Int32  `tfsdk:"token_bucket_rate_limiter_capacity"`
	UseDualstackEndpoint           types.Bool   `tfsdk:"use_dualstack_endpoint"`
	UseFipsEndpoint                types.Bool   `tfsdk:"use_fips_endpoint"`
}

func (p *FastSSMProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "fastssm"
	resp.Version = p.version
}

func (p *FastSSMProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Terraform provider specifically written to overcome the performance limitations I've hit with AWS SSM parameter store in larger deployments.",
		MarkdownDescription: "~> **Note:** When you start having thousands of SSM parameters, you begin to notice quite some slowness of terraform. It's highly likely that you've exhausted the rate limit of AWS SSM API. Even if you upgrade the limit, that only applies for GetParameter calls. In the official AWS provider, for each GetParameter call, there's an additional DescribeParameters call made. That's where the bottleneck is. This provider eliminates >90% of these rate-limited calls, by not doing them in the first place. It's at the expense of not supporting all the metadata, but that should be a fair trade-off, considering you can still use SSM parameter store, fast.",
		Attributes: map[string]schema.Attribute{
			"access_key": schema.StringAttribute{
				Optional: true,
				Description: "The access key for API operations. You can retrieve this\n" +
					"from the 'Security & Credentials' section of the AWS console.",
			},
			"allowed_account_ids": schema.SetAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Validators: []validator.Set{
					setvalidator.ConflictsWith(path.Expressions{
						path.MatchRoot("forbidden_account_ids"),
					}...),
				},
				DeprecationMessage: "This is not supported in this provider intentionally.",
			},
			"assume_role":                   assumeRoleSchema(),
			"assume_role_with_web_identity": assumeRoleWithWebIdentitySchema(),
			"custom_ca_bundle": schema.StringAttribute{
				Optional: true,
				Description: "File containing custom root and intermediate certificates. " +
					"Can also be configured using the `AWS_CA_BUNDLE` environment variable. " +
					"(Setting `ca_bundle` in the shared config file is not supported.)",
			},
			"default_tags": schema.MapAttribute{
				Optional:           true,
				Description:        "Configuration block with settings to default resource tags across all resources.",
				DeprecationMessage: "This is not supported in this provider intentionally.",
				ElementType:        types.StringType,
				// Elem: &schema.Resource{
				// 	Schema: map[string]*schema.Schema{
				// 		"tags": {
				// 			Type:     schema.TypeMap,
				// 			Optional: true,
				// 			Elem:     &schema.Schema{Type: schema.TypeString},
				// 			Description: "Resource tags to default across all resources. " +
				// 				"Can also be configured with environment variables like `" + tftags.DefaultTagsEnvVarPrefix + "<tag_name>`.",
				// 		},
				// 	},
				// },
			},
			"endpoints": endpointsSchema(),
			"forbidden_account_ids": schema.SetAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Description: "Unsupported.",
				Validators: []validator.Set{
					setvalidator.ConflictsWith(path.Expressions{
						path.MatchRoot("allowed_account_ids"),
					}...),
				},
			},
			"http_proxy": schema.StringAttribute{
				Optional: true,
				Description: "URL of a proxy to use for HTTP requests when accessing the AWS API. " +
					"Can also be set using the `HTTP_PROXY` or `http_proxy` environment variables.",
				DeprecationMessage: "This is not supported in this provider intentionally.",
			},
			"https_proxy": schema.StringAttribute{
				Optional: true,
				Description: "URL of a proxy to use for HTTPS requests when accessing the AWS API. " +
					"Can also be set using the `HTTPS_PROXY` or `https_proxy` environment variables.",
				DeprecationMessage: "This is not supported in this provider intentionally.",
			},
			"ignore_tags": schema.ListAttribute{
				Optional: true,
				// 	MaxItems:    1,
				Description:        "Configuration block with settings to ignore resource tags across all resources.",
				DeprecationMessage: "This is not supported in this provider intentionally.",
				ElementType:        types.StringType,
				// 	Elem: &schema.Resource{
				// 		Schema: map[string]*schema.Schema{
				// 			"keys": {
				// 				Type:     schema.TypeSet,
				// 				Optional: true,
				// 				Elem:     &schema.Schema{Type: schema.TypeString},
				// 				Description: "Resource tag keys to ignore across all resources. " +
				// 					"Can also be configured with the " + tftags.IgnoreTagsKeysEnvVar + " environment variable.",
				// 			},
				// 			"key_prefixes": {
				// 				Type:     schema.TypeSet,
				// 				Optional: true,
				// 				Elem:     &schema.Schema{Type: schema.TypeString},
				// 				Description: "Resource tag key prefixes to ignore across all resources. " +
				// 					"Can also be configured with the " + tftags.IgnoreTagsKeyPrefixesEnvVar + " environment variable.",
				// 			},
				// 		},
				// 	},
			},
			"insecure": schema.BoolAttribute{
				Optional: true,
				Description: "Explicitly allow the provider to perform \"insecure\" SSL requests. If omitted, " +
					"default value is `false`",
			},
			"max_retries": schema.Int32Attribute{
				Optional: true,
				Description: "The maximum number of times an AWS API request is\n" +
					"being executed. If the API request still fails, an error is\n" +
					"thrown.",
			},
			"no_proxy": schema.StringAttribute{
				Optional: true,
				Description: "Comma-separated list of hosts that should not use HTTP or HTTPS proxies. " +
					"Can also be set using the `NO_PROXY` or `no_proxy` environment variables.",
				DeprecationMessage: "This is not supported in this provider intentionally.",
			},
			"profile": schema.StringAttribute{
				Optional: true,
				Description: "The profile for API operations. If not set, the default profile\n" +
					"created with `aws configure` will be used.",
			},
			"region": schema.StringAttribute{
				Optional: true,
				Description: "The region where AWS operations will take place. Examples\n" +
					"are us-east-1, us-west-2, etc.", // lintignore:AWSAT003,
			},
			"retry_mode": schema.StringAttribute{
				Optional: true,
				Description: "Specifies how retries are attempted. Valid values are `standard` and `adaptive`. " +
					"Can also be configured using the `AWS_RETRY_MODE` environment variable.",
			},
			"s3_use_path_style": schema.BoolAttribute{
				Optional: true,
				Description: "Set this to true to enable the request to use path-style addressing,\n" +
					"i.e., https://s3.amazonaws.com/BUCKET/KEY. By default, the S3 client will\n" +
					"use virtual hosted bucket addressing when possible\n" +
					"(https://BUCKET.s3.amazonaws.com/KEY). Specific to the Amazon S3 service.",
				DeprecationMessage: "This is not supported in this provider intentionally.",
			},
			// "s3_us_east_1_regional_endpoint": schema.StringAttribute{
			// 	Optional: true,
			// 	Description: "Specifies whether S3 API calls in the `us-east-1` region use the legacy global endpoint or a regional endpoint. " + //lintignore:AWSAT003
			// 		"Valid values are `legacy` or `regional`. " +
			// 		"Can also be configured using the `AWS_S3_US_EAST_1_REGIONAL_ENDPOINT` environment variable or the `s3_us_east_1_regional_endpoint` shared config file parameter",
			// },
			"secret_key": schema.StringAttribute{
				Optional: true,
				Description: "The secret key for API operations. You can retrieve this\n" +
					"from the 'Security & Credentials' section of the AWS console.",
			},
			"shared_config_files": schema.ListAttribute{
				Optional:    true,
				Description: "List of paths to shared config files. If not set, defaults to [~/.aws/config].",
				ElementType: types.StringType,
			},
			"shared_credentials_files": schema.ListAttribute{
				Optional:    true,
				Description: "List of paths to shared credentials files. If not set, defaults to [~/.aws/credentials].",
				ElementType: types.StringType,
			},
			"skip_credentials_validation": schema.BoolAttribute{
				Optional: true,
				Description: "Skip the credentials validation via STS API. " +
					"Used for AWS API implementations that do not have STS available/implemented.",
			},
			"skip_metadata_api_check": schema.BoolAttribute{
				Optional: true,
				Description: "Skip the AWS Metadata API check. " +
					"Used for AWS API implementations that do not have a metadata api endpoint.",
				DeprecationMessage: "This is not supported in this provider intentionally.",
			},
			"skip_region_validation": schema.BoolAttribute{
				Optional: true,
				Description: "Skip static validation of region name. " +
					"Used by users of alternative AWS-like APIs or users w/ access to regions that are not public (yet).",
				DeprecationMessage: "This is not supported in this provider intentionally.",
			},
			"skip_requesting_account_id": schema.BoolAttribute{
				Optional: true,
				Description: "Skip requesting the account ID. " +
					"Used for AWS API implementations that do not have IAM/STS API and/or metadata API.",
				DeprecationMessage: "This is not supported in this provider intentionally.",
			},
			"sts_region": schema.StringAttribute{
				Optional: true,
				Description: "The region where AWS STS operations will take place. Examples\n" +
					"are us-east-1 and us-west-2.", // lintignore:AWSAT003,
				DeprecationMessage: "This is not supported in this provider intentionally.",
			},
			"token": schema.StringAttribute{
				Optional: true,
				Description: "session token. A session token is only required if you are\n" +
					"using temporary security credentials.",
			},
			"token_bucket_rate_limiter_capacity": schema.Int32Attribute{
				Optional:           true,
				Description:        "The capacity of the AWS SDK's token bucket rate limiter.",
				DeprecationMessage: "This is not supported in this provider intentionally.",
			},
			"use_dualstack_endpoint": schema.BoolAttribute{
				Optional:           true,
				Description:        "Resolve an endpoint with DualStack capability",
				DeprecationMessage: "This is not supported in this provider intentionally.",
			},
			"use_fips_endpoint": schema.BoolAttribute{
				Optional:           true,
				Description:        "Resolve an endpoint with FIPS capability",
				DeprecationMessage: "This is not supported in this provider intentionally.",
			},
		},
	}
}

func endpointsSchema() *schema.SetNestedAttribute {
	return &schema.SetNestedAttribute{
		Optional: true,
		NestedObject: schema.NestedAttributeObject{
			Attributes: map[string]schema.Attribute{
				"ssm": schema.StringAttribute{
					Optional:    true,
					Description: "Use this to override the default service endpoint URL",
				},
			},
		},
	}
}

func assumeRoleSchema() *schema.ListNestedAttribute {
	return &schema.ListNestedAttribute{
		Optional: true,
		NestedObject: schema.NestedAttributeObject{
			Attributes: map[string]schema.Attribute{
				"duration": schema.StringAttribute{
					Optional:    true,
					Description: "The duration, between 15 minutes and 12 hours, of the role session. Valid time units are ns, us (or µs), ms, s, h, or m.",
					Validators: []validator.String{
						durationValidator{},
					},
				},
				"external_id": schema.StringAttribute{
					Optional:    true,
					Description: "A unique identifier that might be required when you assume a role in another account.",
					Validators: []validator.String{
						stringvalidator.LengthBetween(2, 1024),
						stringvalidator.RegexMatches(regexache.MustCompile(`[\w+=,.@:\/\-]*`), ""),
					},
				},
				"policy": schema.StringAttribute{
					Optional:    true,
					Description: "IAM Policy JSON describing further restricting permissions for the IAM Role being assumed.",
					Validators: []validator.String{
						jsonValidator{},
					},
				},
				"policy_arns": schema.SetAttribute{
					Optional:    true,
					Description: "Amazon Resource Names (ARNs) of IAM Policies describing further restricting permissions for the IAM Role being assumed.",
					ElementType: types.StringType,
					Validators: []validator.Set{
						arnValidator{kind: "set"},
					},
				},
				"role_arn": schema.StringAttribute{
					Optional:    true, // For historical reasons, we allow an empty `assume_role` block
					Description: "Amazon Resource Name (ARN) of an IAM Role to assume prior to making API calls.",
					Validators: []validator.String{
						arnValidator{kind: "string"},
					},
				},
				"session_name": schema.StringAttribute{
					Optional:    true,
					Description: "An identifier for the assumed role session.",
					Validators: []validator.String{
						stringvalidator.All(
							stringvalidator.LengthBetween(2, 64),
							stringvalidator.RegexMatches(regexache.MustCompile(`[\w+=,.@\-]*`), ""),
						),
					},
				},
				"source_identity": schema.StringAttribute{
					Optional:    true,
					Description: "Source identity specified by the principal assuming the role.",
					Validators: []validator.String{
						stringvalidator.All(
							stringvalidator.LengthBetween(2, 64),
							stringvalidator.RegexMatches(regexache.MustCompile(`[\w+=,.@\-]*`), ""),
						),
					},
				},
				"tags": schema.MapAttribute{
					Optional:    true,
					Description: "Assume role session tags.",
					ElementType: types.StringType,
				},
				"transitive_tag_keys": schema.SetAttribute{
					Optional:    true,
					Description: "Assume role session tag keys to pass to any subsequent sessions.",
					ElementType: types.StringType,
				},
			},
		},
	}
}

func assumeRoleWithWebIdentitySchema() *schema.ListNestedAttribute {
	return &schema.ListNestedAttribute{
		Optional: true,
		Validators: []validator.List{
			listvalidator.SizeAtMost(1),
		},
		NestedObject: schema.NestedAttributeObject{
			Attributes: map[string]schema.Attribute{
				"duration": schema.StringAttribute{
					Optional:    true,
					Description: "The duration, between 15 minutes and 12 hours, of the role session. Valid time units are ns, us (or µs), ms, s, h, or m.",
					Validators: []validator.String{
						durationValidator{},
					},
				},
				"policy": schema.StringAttribute{
					Optional:    true,
					Description: "IAM Policy JSON describing further restricting permissions for the IAM Role being assumed.",
					Validators: []validator.String{
						jsonValidator{},
					},
				},
				"policy_arns": schema.SetAttribute{
					Optional:    true,
					Description: "Amazon Resource Names (ARNs) of IAM Policies describing further restricting permissions for the IAM Role being assumed.",
					ElementType: types.StringType,
					Validators: []validator.Set{
						arnValidator{kind: "set"},
					},
				},
				"role_arn": schema.StringAttribute{
					Optional:    true, // For historical reasons, we allow an empty `assume_role_with_web_identity` block
					Description: "Amazon Resource Name (ARN) of an IAM Role to assume prior to making API calls.",
					Validators: []validator.String{
						arnValidator{kind: "string"},
					},
				},
				"session_name": schema.StringAttribute{
					Optional:    true,
					Description: "An identifier for the assumed role session.",
					Validators: []validator.String{
						stringvalidator.All(
							stringvalidator.LengthBetween(2, 64),
							stringvalidator.RegexMatches(regexache.MustCompile(`[\w+=,.@\-]*`), ""),
						),
					},
				},
				"web_identity_token": schema.StringAttribute{
					Optional: true,
					// ExactlyOneOf: []string{"assume_role_with_web_identity.0.web_identity_token", "assume_role_with_web_identity.0.web_identity_token_file"},
					Validators: []validator.String{
						stringvalidator.All(
							stringvalidator.LengthBetween(4, 20000),
							stringvalidator.ConflictsWith(path.Expressions{
								path.MatchRoot("web_identity_token_file"),
							}...),
						),
					},
				},
				"web_identity_token_file": schema.StringAttribute{
					Optional: true,
					// ExactlyOneOf: []string{"assume_role_with_web_identity.0.web_identity_token", "assume_role_with_web_identity.0.web_identity_token_file"},
					Validators: []validator.String{
						stringvalidator.All(
							stringvalidator.ConflictsWith(path.Expressions{
								path.MatchRoot("web_identity_token"),
							}...),
						),
					},
				},
			},
		},
	}
}

func (p *FastSSMProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	// Retrieve provider data from configuration
	var data FastSSMProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var options = []func(*config.LoadOptions) error{}

	if !data.RetryMode.IsNull() {
		mode, err := aws.ParseRetryMode(data.RetryMode.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"parse retry_mode failed",
				err.Error(),
			)

		}
		options = append(options, config.WithRetryMode(mode), config.WithRetryMaxAttempts(25))
	}

	// Region
	if !data.Region.IsNull() {
		options = append(options, config.WithDefaultRegion(data.Region.ValueString()))
	}

	// AWS Profile
	if !data.Profile.IsNull() {
		options = append(options, config.WithSharedConfigProfile(data.Profile.ValueString()))
	}

	// Static credentials
	if !data.AccessKey.IsNull() && !data.SecretKey.IsNull() {
		creds := staticCredentials{
			accessKey: data.AccessKey.ValueString(),
			secretKey: data.SecretKey.ValueString(),
		}
		if !data.Token.IsNull() {
			creds.token = data.Token.ValueString()
		}

		options = append(options, config.WithCredentialsProvider(creds))
	}

	// TODO add assumerole support
	// config.WithAssumeRoleCredentialOptions()
	// config.WithSharedCredentialsFiles()

	// TODO add web-identity-role support
	// if !data.AssumeRoleWithWebIdentity.IsNull() {
	// 	options = append(options, config.WithWebIdentityRoleCredentialOptions(func(*stscreds.WebIdentityRoleOptions)))
	// }

	// Client configuration for data sources and resources
	cfg, err := config.LoadDefaultConfig(context.TODO(), options...)
	if err != nil {
		resp.Diagnostics.AddError(
			"provider configuration failed",
			err.Error(),
		)
		return
	}

	stsclient := sts.NewFromConfig(cfg)
	res, err := stsclient.GetCallerIdentity(context.TODO(), &sts.GetCallerIdentityInput{})
	if err != nil || res == nil {
		resp.Diagnostics.AddError(
			"provider configuration failed at STS GetCallerIdentity phase",
			err.Error(),
		)
		return
	}

	if res.UserId == nil {
		resp.Diagnostics.AddError(
			"couldn't get through STS authentication",
			"Validation of credentials against STS failed. The response from AWS contained no userID.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	client := ssm.NewFromConfig(cfg)
	resp.DataSourceData = client
	resp.ResourceData = client
}

type staticCredentials struct {
	accessKey string
	secretKey string
	token     string
}

func (s staticCredentials) Retrieve(context.Context) (aws.Credentials, error) {
	return aws.Credentials{
		AccessKeyID:     s.accessKey,
		SecretAccessKey: s.secretKey,
		SessionToken:    s.token,
	}, nil
}

func (p *FastSSMProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewParameterResource,
	}
}

func (p *FastSSMProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewParameterDataSource,
	}
}

// func (p *FastSSMProvider) Functions(ctx context.Context) []func() function.Function {
// 	return []func() function.Function{
// 		NewExampleFunction,
// 	}
// }

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &FastSSMProvider{
			version: version,
		}
	}
}
