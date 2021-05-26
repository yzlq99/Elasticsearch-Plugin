# ElasticsearchPlugin
elasticsearch plugin

skip list

参考：
https://medium.com/tinder-engineering/how-we-improved-our-performance-using-elasticsearch-plugins-part-2-b051da2ee85b

## GO RoaringBitmap
```go
package main

import (
	"bytes"
	"fmt"

	"github.com/RoaringBitmap/roaring"
)

func main() {

	rb := roaring.BitmapOf(3, 4, 100, 200)
	buf := new(bytes.Buffer)
	_, err := rb.WriteTo(buf)
	if err != nil {
		fmt.Errorf("Failed writing")
	}

	fmt.Printf("%v\n", buf.Bytes())

	rb64, _ := rb.ToBase64()

	fmt.Println(rb64)

	newrb := roaring.New()
	_, err = newrb.FromBase64(rb64)
	if err != nil {
		fmt.Errorf("Failed reading")
	}
	if !rb.Equals(newrb) {
		fmt.Errorf("Cannot retrieve serialized version")
	}

	fmt.Println(rb.Contains(1))
	fmt.Println(newrb.Contains(1))

	fmt.Println(newrb.String())

}
```

## Java RoaringBitmap
```java
package org.test.skiplist;

import java.io.IOException;
import java.nio.ByteBuffer;
import java.util.Base64;

import org.roaringbitmap.RoaringBitmap;
import org.roaringbitmap.buffer.MutableRoaringBitmap;

public class BitMap {

	public static void main(String[] args) {
		MutableRoaringBitmap mrb = MutableRoaringBitmap.bitmapOf(3, 4, 100);
		System.out.println("starting with  bitmap " + mrb);
		ByteBuffer outbb = ByteBuffer.allocate(mrb.serializedSizeInBytes());
		mrb.serialize(outbb);
		outbb.flip();
		String serializedstring = Base64.getEncoder().encodeToString(outbb.array());
		System.out.println("serializedstring :\n" + serializedstring);

		String a = "OjAAAAEAAAAAAAMAEAAAAAMABABkAMgA";
		byte[] bt = Base64.getDecoder().decode(a);

		ByteBuffer buffer = ByteBuffer.wrap(bt);
		RoaringBitmap ret = new RoaringBitmap();
		try {
			ret.deserialize(buffer);
		} catch (IOException ioe) {
			System.out.println(ioe.getMessage());
			return;
		}

		System.out.println(ret.toString());
	}

}
```

## ES Test Query
```json
GET institution/_search
{
  "query": {
    "function_score": {
      "query": {
        "match_all": {}
      },
      "boost_mode": "multiply", 
      "functions": [
        {
          "script_score": {
            "script": {
              "params": {
                "skip": "OjAAAAEAAAAAAAMAEAAAAAMABQBkAMgA"
              }, 
              "lang": "skip_list",
              "source": "pure_df"
            }
          }
        }
      ]
    }
  }
}


DELETE institution


GET institution


PUT institution
{
  "mappings": {
    "dynamic_templates": [
      {
        "ik_fields": {
          "path_match": "ik.*",
          "match_mapping_type": "string",
          "mapping": {
            "analyzer": "ik_max_word",
            "search_analyzer": "ik_smart",
            "type": "text"
          }
        }
      },
      {
        "keyword_fields": {
          "path_match": "kw.*",
          "match_mapping_type": "string",
          "mapping": {
            "analyzer": "standard",
            "type": "keyword"
          }
        }
      }
    ]
  }
}

PUT institution/_doc/5
{
        "kw.id": "5",
        "kw.entity_type": 109006022,
        "kw.founded_year": "2009"
}

PUT institution/_doc/3
{
        "kw.id": "3",
        "kw.entity_type": 109006022,
        "kw.founded_year": "2009"
}
```
