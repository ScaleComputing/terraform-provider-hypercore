package unit

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/schema"
	"github.com/stretchr/testify/assert"
)

// Test Schema function of HypercoreISOResource
func TestHypercoreISOResource_Schema(t *testing.T) {
	// Create an instance of the resource
	r := &HypercoreISOResource{}

	// Prepare request and response objects
	req := resource.SchemaRequest{}
	resp := &resource.SchemaResponse{}

	// Call the Schema function
	r.Schema(context.Background(), req, resp)

	// Validate the schema is set
	assert.NotNil(t, resp.Schema)
	assert.NotNil(t, resp.Schema.Attributes)

	// Check the description
	assert.Contains(t, resp.Schema.MarkdownDescription, "Hypercore ISO resource to manage ISO images")

	// Check individual attributes
	attributes := resp.Schema.Attributes

	// Check ID attribute
	idAttr, ok := attributes["id"].(schema.StringAttribute)
	assert.True(t, ok)
	assert.True(t, idAttr.Computed)
	assert.Contains(t, idAttr.MarkdownDescription, "ISO identifier")

	// Check Name attribute
	nameAttr, ok := attributes["name"].(schema.StringAttribute)
	assert.True(t, ok)
	assert.True(t, nameAttr.Required)
	assert.Contains(t, nameAttr.MarkdownDescription, "Desired name of the ISO to upload")

	// Check Source URL attribute
	sourceURLAttr, ok := attributes["source_url"].(schema.StringAttribute)
	assert.True(t, ok)
	assert.True(t, sourceURLAttr.Optional)
	assert.Contains(t, sourceURLAttr.MarkdownDescription, "Source URL from where to fetch")
}
