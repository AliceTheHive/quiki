<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8" />
{{with .Description}}
    <meta name="description" content="{{.}}" />
{{end}}
{{with .KeywordString}}
    <meta name="keywords" content="{{.}}" />
{{end}}
{{with .Author}}
    <meta name="author" content="{{.}}" />
{{end}}
    <title>{{.VisibleTitle}}</title>
    <link rel="stylesheet" type="text/css" href="{{.StaticRoot}}/style.css" />
    <link rel="stylesheet" type="text/css" href="/static/quiki.css" />
{{with .PageCSS}}
    <style>
{{.}}
    </style>
{{end}}
{{range .Scripts}}
    <script src="{{.}}"></script>
{{end}}
</head>

<body>
<div id="container">

    <div id="header">
        <ul id="navigation">
            {{range .Navigation}}
                <li><a href="{{.Link}}">{{.Display}}</a></li>
            {{end}}
        </ul>
        <a href="{{.Root.Wiki}}/">
            {{if .WikiLogo}}
                <img src="{{.WikiLogo}}" alt="{{.WikiTitle}}" data-rjs="3" />
            {{else}}
                <h1>{{.WikiTitle}}</h1>
            {{end}}
        </a>
    </div>

    <div id="content">
