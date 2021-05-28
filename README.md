# ElasticsearchPlugin

一个 elasticsearch 的 script plugin。
实现的逻辑是从参数中接收一个 base64 编码的 bitmap，然后判断文档 id 是否存在于 bitmap 中，如果存在则脚本打分为 0，否则打分为 1。配合 boost_mode 可以实现文档最终评分为 0 或者保持现状。
另外配合 min_score 可以实现跳过分数为 0 文档。
对应实际业务逻辑即看过的文档不再出现第二遍。

参考：
https://medium.com/tinder-engineering/how-we-improved-our-performance-using-elasticsearch-plugins-part-2-b051da2ee85b

## 添加 Plugin 步骤
1. ExpertScriptPlugin 类编码
2. maven clean
3. maven install
4. 复制 tar 包到 es-docker 下
5. 修改 plugin-descriptor.properties （如果改了类名或者 plugin 的名字）
6. 压缩成 zip（zip 名字和 Dockerfile 中对应）
7. 执行 docker build -f ./Dockerfile -t yinzl/elasticsearch .
8. 重启 Elasticsearch

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
  
  // 序列化为 base64 编码字符串
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

		byte[] bt = Base64.getDecoder().decode(serializedstring);

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
                "skip": "OjAAAAEAAAAAAAMAEAAAAAMABQBkAMgA",
                // 这个字段名对应的字段一定要是 keyword 字段
                "field_name": "kw.id"
              }, 
              "lang": "expert_scripts",
              "source": "skip_list"
            }
          }
        }
      ]
    }
  }
}

// mapping
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

PUT institution/_doc/4
{
        "kw.id": "4",
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
