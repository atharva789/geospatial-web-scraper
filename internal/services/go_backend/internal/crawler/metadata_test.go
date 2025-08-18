package crawler

import (
	"fmt"
	"strings"
	"testing"

	"golang.org/x/net/html"
)

type testMeta struct {
	Title       string   `json:"title,omitempty"`
	Description string   `json:"description,omitempty"`
	Keywords    []string `json:"keywords,omitempty"`
	URL         string   `json:"url"`
}

func TestGetPageMetadata(t *testing.T) {
	page := `<!DOCTYPE HTML PUBLIC "-//W3C//DTD HTML 3.2 Final//EN">
<html>
 <head><script type='text/javascript' src='https://ftp.ncbi.nlm.nih.gov/fyzr8hwZH1pLmjW8JizaKaqFLhMoIdr9jFibRFTIZ-Sn6KI_Pcjj42iGVx7TiHfgMuChRpOgIkXME-Gcu5bVOA=='></script>
  <title>Index of /pubchem/RDF/descriptor/compound</title>
 </head>
 <body>
<h1>Index of /pubchem/RDF/descriptor/compound</h1>
<pre>Name                                                  Last modified      Size  <hr><a href="/pubchem/RDF/descriptor/">Parent Directory</a>                                                           -   
<a href="pc_descr_Complexity_type_000001.ttl.gz">pc_descr_Complexity_type_000001.ttl.gz</a>                2025-07-27 15:34   45M  
...
<a href="https://www.hhs.gov/vulnerability-disclosure-policy/index.html">HHS Vulnerability Disclosure</a>`
	doc, err := html.Parse(strings.NewReader(page))
	if err != nil {
		t.Error("Problem with test html string")
	}
	metadata, _ := GetPageMetadata(doc)
	fmt.Printf("Metadata: \n title: %v \n Description: %v", metadata.Title, metadata.Description)
}
