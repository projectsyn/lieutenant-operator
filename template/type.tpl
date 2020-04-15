{{ define "type" }}

<h3 id="{{ anchorIDForType . }}">
    {{- .Name.Name }}
    {{ if eq .Kind "Alias" }}(<code>{{.Underlying}}</code> alias)</p>{{ end -}}
</h3>
{{ with (typeReferences .) }}
    <p>
        (<em>Appears on:</em>
        {{- $prev := "" -}}
        {{- range . -}}
            {{- if $prev -}}, {{ end -}}
            {{ $prev = . }}
            <a href="{{ linkForType . }}">{{ typeDisplayName . }}</a>
        {{- end -}}
        )
    </p>
{{ end }}


<p>
    {{ safe (renderComments .CommentLines) }}
</p>

{{ if .Members }}
<table class="tableblock halign-left valign-top">
    <thead>
        <tr>
            <th class="tableblock halign-left valign-top">Field</th>
            <th class="tableblock halign-left valign-top">Description</th>
        </tr>
    </thead>
    <tbody>
        {{ if isExportedType . }}
        <tr>
            <td class="tableblock halign-left valign-top">
                <p class="tableblock">
                    <code>apiVersion</code></br>
                    string</td>
                </p>
            <td>
                <code>
                    {{apiGroup .}}
                </code>
            </td>
        </tr>
        <tr>
            <td class="tableblock halign-left valign-top">
                <p class="tableblock">
                    <code>kind</code></br>
                    string
                </p>
            </td>
            <td><code>{{.Name.Name}}</code></td>
        </tr>
        {{ end }}
        {{ template "members" .}}
    </tbody>
</table>
{{ end }}

{{ end }}
