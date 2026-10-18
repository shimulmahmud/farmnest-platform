import sys
with open(sys.argv[1], 'r') as f:
    content = f.read()
old = '                {\n                        Keys:    bson.D{{Key: "name", Value: "text"}},\n                        Options: options.Index().SetName("idx_product_name_text"),\n                },\n        }'
new = '                {\n                        Keys: bson.D{\n                                {Key: "name", Value: "text"},\n                                {Key: "brand", Value: "text"},\n                                {Key: "description", Value: "text"},\n                        },\n                        Options: options.Index().SetName("idx_product_search").\n                                SetWeights([]float64{10, 5, 3}),\n                },\n                {\n                        Keys:    bson.D{{Key: "variants.sku", Value: "text"}},\n                        Options: options.Index().SetName("idx_product_sku_text"),\n                },\n        }'
content = content.replace(old, new)
with open(sys.argv[1], 'w') as f:
    f.write(content)
print('Done')
