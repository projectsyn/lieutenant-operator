{{ define "members" }}

{{ range .Members }}
{{ if not (hiddenMember .)}}
<tr>
    <td class="tableblock halign-left valign-top">
        <p class="tableblock">
            <code>{{ fieldName . }}</code></br>
            <em>
                {{ if linkForType .Type }}
                    <a href="{{ linkForType .Type}}">
                        {{ typeDisplayName .Type }}
                    </a>
                {{ else }}
                    {{ typeDisplayName .Type }}
                {{ end }}
            </em>
        </p>
    </td>
    <td class="tableblock halign-left valign-top">
        <p class="tableblock">
            {{ if fieldEmbedded . }}
                <p>
                    (Members of <code>{{ fieldName . }}</code> are embedded into this type.)
                </p>
            {{ end}}

            {{ if isOptionalMember .}}
                <em>(Optional)</em>
            {{ end }}

            {{ safe (renderComments .CommentLines) }}

            {{ if and (eq (.Type.Name.Name) "ObjectMeta") }}
                Refer to the Kubernetes API documentation for the fields of the
                <code>metadata</code> field.
            {{ end }}

            {{ if or (eq (fieldName .) "spec") }}
                <br/>
                <br/>
                <table class="tableblock frame-all grid-all stretch">
                    {{ template "members" .Type }}
                </table>
            {{ end }}
        <p class="tableblock">
    </td>
</tr>
{{ end }}
{{ end }}

{{ end }}
