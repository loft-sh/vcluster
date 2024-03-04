package util

const TemplatePartialImport = `
import %s from "%s"`
const TemplatePartialUse = `<%s />
`
const TemplatePartialUseConfig = `<%s />
`
const TemplateConfigField = `
<details className="config-field" data-expandable="%t"%s>
<summary>

%s` + "`%s`" + ` <span className="config-field-required" data-required="%t">required</span> <span className="config-field-type">%s</span> <span className="config-field-default">%s</span> <span className="config-field-enum">%s</span> {#%s}

%s

</summary>

%s

</details>
`
const TemplateFunctionRef = `
<details className="config-field -function" data-expandable="%t"%s>
<summary>

%s` + "`%s`" + ` <span className="config-field-type">%s</span> <span className="config-field-enum">%s</span> <span className="config-field-default -return">%s</span> <span className="config-field-required" data-required="%t">pipeline only</span>  {#%s}

%s

</summary>

%s

</details>
`
