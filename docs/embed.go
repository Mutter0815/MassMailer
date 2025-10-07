package docs

import _ "embed"

//go:embed campaign-api.openapi.yaml
var embeddedCampaignOpenAPI []byte

//go:embed swagger.html
var embeddedCampaignSwaggerHTML []byte

// CampaignOpenAPI содержит OpenAPI-спецификацию кампаний.
var CampaignOpenAPI = embeddedCampaignOpenAPI

// CampaignSwaggerHTML содержит HTML-страницу с Swagger UI.
var CampaignSwaggerHTML = embeddedCampaignSwaggerHTML
