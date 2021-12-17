package GoMybatis

import (
	"bytes"
	"github.com/timandy/GoMybatis/lib/github.com/beevik/etree"
	"github.com/timandy/GoMybatis/utils"
	"reflect"
	"strconv"
	"strings"
)

var equalOperator = []string{"/", "+", "-", "*", "**", "|", "^", "&", "%", "<", ">", ">=", "<=", " in ", " not in ", " or ", "||", " and ", "&&", "==", "!="}

/**
TODO sqlTemplate解析器，目前直接操作*etree.Element实现，后期应该改成操作xml，换取更好的维护性
*/
type GoMybatisTemplateDecoder struct {
	print bool
}

type LogicDeleteData struct {
	Column   string
	Property string
	LangType string

	Enable         bool
	Deleted_value  string
	Undelete_value string
}

type VersionData struct {
	Column   string
	Property string
	LangType string
}

func (it *GoMybatisTemplateDecoder) SetPrintElement(print bool) {
	it.print = print
}

func (it *GoMybatisTemplateDecoder) DecodeTree(tree map[string]etree.Token, beanType reflect.Type) error {
	if tree == nil {
		return utils.NewError("GoMybatisTemplateDecoder", "decode data map[string]*MapperXml cant be nil!")
	}
	if beanType != nil {
		if beanType.Kind() == reflect.Ptr {
			beanType = beanType.Elem()
		}
	}
	for _, item := range tree {
		var typeString = reflect.TypeOf(item).String()
		if typeString == "*etree.Element" {
			var v = item.(*etree.Element)
			var method *reflect.StructField
			if beanType != nil {
				if isMethodElement(v.Tag) {
					var upperId = utils.UpperFieldFirstName(v.SelectAttrValue("id", ""))
					if upperId == "" {
						upperId = utils.UpperFieldFirstName(v.Tag)
					}
					m, haveMethod := beanType.FieldByName(upperId)
					if haveMethod {
						method = &m
					}
				}
			}
			var oldChilds = v.Child
			v.Child = []etree.Token{}
			var newTree = v
			var success, _ = it.Decode(method, newTree, tree)
			newTree.Child = append(newTree.Child, oldChilds...)
			*v = *newTree

			//println
			if success {
				var beanName string
				if beanType != nil {
					beanName = beanType.String()
				}
				if it.print {
					var s = "================DecoderTemplate " + beanName + "." + v.SelectAttrValue("id", "") + "============\n"
					printElement(v, &s)
					println(s)
				}
			}
		}
	}
	return nil
}

func printElement(element *etree.Element, v *string) {
	*v += "<" + element.Tag + " "
	for _, item := range element.Attr {
		*v += item.Key + "=\"" + item.Value + "\""
	}
	*v += " >"
	if element.Child != nil && len(element.Child) != 0 {
		for _, item := range element.Child {
			var typeString = reflect.TypeOf(item).String()
			if typeString == "*etree.Element" {
				var nStr = ""
				printElement(item.(*etree.Element), &nStr)
				*v += nStr
			} else if typeString == "*etree.CharData" {
				*v += "" + item.(*etree.CharData).Data
			}
		}
	}
	*v += "</" + element.Tag + ">\n"
}

func (it *GoMybatisTemplateDecoder) Decode(method *reflect.StructField, mapper *etree.Element, tree map[string]etree.Token) (bool, error) {

	switch mapper.Tag {

	case "selectTemplate":
		mapper.Tag = Element_Select

		var id = mapper.SelectAttrValue("id", "")
		if id == "" {
			mapper.CreateAttr("id", "selectTemplate")
		}

		var tables = mapper.SelectAttrValue("tables", "")
		var columns = mapper.SelectAttrValue("columns", "")
		var wheres = mapper.SelectAttrValue("wheres", "")

		var resultMap = mapper.SelectAttrValue("resultMap", "")
		if resultMap == "" {
			resultMap = "BaseResultMap"
		}
		var resultMapData = tree[resultMap].(*etree.Element)
		if resultMapData == nil {
			panic(utils.NewError("GoMybatisTemplateDecoder", "resultMap not define! id = ", resultMap))
		}
		checkTablesValue(mapper, &tables, resultMapData)

		var logic = it.decodeLogicDelete(resultMapData)

		var sql bytes.Buffer
		sql.WriteString("select ")
		if columns == "" {
			columns = "*"
		}
		sql.WriteString(columns)
		sql.WriteString(" from ")
		sql.WriteString(tables)
		if len(wheres) > 0 {
			//sql.WriteString(" where ")
			mapper.Child = append(mapper.Child, &etree.CharData{
				Data: sql.String(),
			})
			//TODO decode wheres
			it.DecodeWheres(wheres, mapper, logic, nil)
		}
		break
	case "insertTemplate": //已支持批量
		mapper.Tag = Element_Insert

		var id = mapper.SelectAttrValue("id", "")
		if id == "" {
			mapper.CreateAttr("id", "insertTemplate")
		}

		var tables = mapper.SelectAttrValue("tables", "")
		var inserts = mapper.SelectAttrValue("inserts", "")

		var resultMap = mapper.SelectAttrValue("resultMap", "")
		if resultMap == "" {
			resultMap = "BaseResultMap"
		}
		if inserts == "" {
			inserts = "*?*"
		}

		var resultMapData = tree[resultMap].(*etree.Element)
		if resultMapData == nil {
			panic(utils.NewError("GoMybatisTemplateDecoder", "resultMap not define! id = ", resultMap))
		}
		checkTablesValue(mapper, &tables, resultMapData)

		var logic = it.decodeLogicDelete(resultMapData)

		var collectionName = it.DecodeCollectionName(method)

		//start builder
		var sql bytes.Buffer
		sql.WriteString("insert into ")
		sql.WriteString(tables)

		mapper.Child = append(mapper.Child, &etree.CharData{
			Data: sql.String(),
		})

		//add insert column
		var trimColumn = etree.Element{
			Tag: Element_Trim,
			Attr: []etree.Attr{
				{Key: "prefix", Value: "("},
				{Key: "suffix", Value: ")"},
				{Key: "suffixOverrides", Value: ","},
			},
			Child: []etree.Token{},
		}

		//cloumns
		if collectionName != "" {
			for _, v := range resultMapData.ChildElements() {
				if inserts == "*" || inserts == "*?*" {
					trimColumn.Child = append(trimColumn.Child, &etree.CharData{
						Data: v.SelectAttrValue("column", "") + ",",
					})
				}
			}
		} else {
			for _, v := range resultMapData.ChildElements() {
				if collectionName == "" && inserts == "*?*" {
					trimColumn.Child = append(trimColumn.Child, &etree.Element{
						Tag: Element_If,
						Attr: []etree.Attr{
							{Key: "test", Value: it.makeIfNotNull(v.SelectAttrValue("column", ""))},
						},
						Child: []etree.Token{
							&etree.CharData{
								Data: v.SelectAttrValue("column", "") + ",",
							},
						},
					})
				} else if inserts == "*" {
					trimColumn.Child = append(trimColumn.Child, &etree.CharData{
						Data: v.SelectAttrValue("column", "") + ",",
					})
				}
			}
		}

		mapper.Child = append(mapper.Child, &trimColumn)

		//args
		var tempElement = etree.Element{
			Tag:   Element_Trim,
			Attr:  []etree.Attr{{Key: "prefix", Value: "values ("}, {Key: "suffix", Value: ")"}, {Key: "suffixOverrides", Value: ","}},
			Child: []etree.Token{},
		}

		if collectionName == "" {
			for _, v := range resultMapData.ChildElements() {
				if logic.Enable && v.SelectAttrValue("column", "") == logic.Property {
					tempElement.Child = append(tempElement.Child, &etree.CharData{
						Data: logic.Undelete_value + ",",
					})
					continue
				}
				if inserts == "*?*" {
					tempElement.Child = append(tempElement.Child, &etree.Element{
						Tag:  Element_If,
						Attr: []etree.Attr{{Key: "test", Value: it.makeIfNotNull(v.SelectAttrValue("column", ""))}},
						Child: []etree.Token{
							&etree.CharData{
								Data: "#{" + v.SelectAttrValue("column", "") + "},",
							},
						},
					})
				} else if inserts == "*" {
					tempElement.Child = append(tempElement.Child, &etree.CharData{
						Data: "#{" + v.SelectAttrValue("column", "") + "},",
					})
				}
			}
		} else {
			tempElement.Attr = []etree.Attr{}
			tempElement.Tag = Element_Foreach
			tempElement.Attr = []etree.Attr{{Key: "open", Value: "values "}, {Key: "close", Value: ""}, {Key: "separator", Value: ","}, {Key: "collection", Value: collectionName}}
			tempElement.Child = []etree.Token{}
			for index, v := range resultMapData.ChildElements() {
				var prefix = ""
				if index == 0 {
					prefix = "("
				}
				//TODO serch property
				var defProperty = v.SelectAttrValue("column", "")
				if method != nil {
					for i := 0; i < method.Type.NumIn(); i++ {
						var argItem = method.Type.In(i)
						if argItem.Kind() == reflect.Ptr {
							argItem = argItem.Elem()
						}
						if argItem.Kind() == reflect.Slice || argItem.Kind() == reflect.Array {
							argItem = argItem.Elem()
						}
						if argItem.Kind() == reflect.Struct {
							for k := 0; k < argItem.NumField(); k++ {
								var argStructField = argItem.Field(k)
								var js = argStructField.Tag.Get("json") //扫描json tag
								if strings.Index(js, ",") != -1 {
									js = strings.Split(js, ",")[0]
								}
								if strings.ToLower(strings.Replace(defProperty, "_", "", -1)) ==
									strings.ToLower(strings.Replace(argStructField.Name, "_", "", -1)) ||
									js == defProperty {
									defProperty = argStructField.Name
								}
							}
						}
					}
				}
				var value = prefix + "#{" + "item." + defProperty + "}"
				if logic.Enable && v.SelectAttrValue("column", "") == logic.Property {
					value = `'` + logic.Undelete_value + "'"
				}
				if index+1 == len(resultMapData.ChildElements()) {
					value += ")"
				} else {
					value += ","
				}
				tempElement.Child = append(tempElement.Child, &etree.CharData{
					Data: value,
				})
			}
		}
		mapper.Child = append(mapper.Child, &tempElement)

		break
	case "updateTemplate":
		mapper.Tag = Element_Update

		var id = mapper.SelectAttrValue("id", "")
		if id == "" {
			mapper.CreateAttr("id", "updateTemplate")
		}

		var tables = mapper.SelectAttrValue("tables", "")
		var columns = mapper.SelectAttrValue("sets", "")
		var wheres = mapper.SelectAttrValue("wheres", "")

		var resultMap = mapper.SelectAttrValue("resultMap", "")
		if resultMap == "" {
			resultMap = "BaseResultMap"
		}

		var resultMapData = tree[resultMap].(*etree.Element)
		if resultMapData == nil {
			panic(utils.NewError("GoMybatisTemplateDecoder", "resultMap not define! id = ", resultMap))
		}
		checkTablesValue(mapper, &tables, resultMapData)

		var logic = it.decodeLogicDelete(resultMapData)

		var versionData = it.decodeVersionData(resultMapData)

		var sql bytes.Buffer
		sql.WriteString("update ")
		sql.WriteString(tables)
		sql.WriteString(" set ")
		if columns == "" {
			mapper.Child = append(mapper.Child, &etree.CharData{
				Data: sql.String(),
			})
			sql.Reset()
			for _, v := range resultMapData.ChildElements() {
				if v.Tag == "id" {

				} else {
					if v.SelectAttrValue("version_enable", "") == "true" {
						continue
					}
					columns += v.SelectAttrValue("column", "") + "?" + v.SelectAttrValue("column", "") + " = #{" + v.SelectAttrValue("column", "") + "},"
				}
			}
			columns = strings.Trim(columns, ",")
			it.DecodeSets(columns, mapper, LogicDeleteData{}, versionData)
		} else {
			mapper.Child = append(mapper.Child, &etree.CharData{
				Data: sql.String(),
			})
			sql.Reset()
			it.DecodeSets(columns, mapper, LogicDeleteData{}, versionData)
		}
		if len(wheres) > 0 || logic.Enable {
			//sql.WriteString(" where ")
			mapper.Child = append(mapper.Child, &etree.CharData{
				Data: sql.String(),
			})
			it.DecodeWheres(wheres, mapper, logic, versionData)
		}
		break
	case "deleteTemplate":
		mapper.Tag = Element_Delete

		var id = mapper.SelectAttrValue("id", "")
		if id == "" {
			mapper.CreateAttr("id", "deleteTemplate")
		}

		var tables = mapper.SelectAttrValue("tables", "")
		var wheres = mapper.SelectAttrValue("wheres", "")

		var resultMap = mapper.SelectAttrValue("resultMap", "")
		if resultMap == "" {
			resultMap = "BaseResultMap"
		}

		var resultMapData = tree[resultMap].(*etree.Element)
		if resultMapData == nil {
			panic(utils.NewError("GoMybatisTemplateDecoder", "resultMap not define! id = ", resultMap))
		}
		checkTablesValue(mapper, &tables, resultMapData)

		var logic = it.decodeLogicDelete(resultMapData)
		if logic.Enable {
			//enable logic delete
			var sql bytes.Buffer
			sql.WriteString("update ")
			sql.WriteString(tables)
			sql.WriteString(" set ")
			mapper.Child = append(mapper.Child, &etree.CharData{
				Data: sql.String(),
			})
			sql.Reset()
			it.DecodeSets("", mapper, logic, nil)
			if len(wheres) > 0 {
				//sql.WriteString(" where ")
				mapper.Child = append(mapper.Child, &etree.CharData{
					Data: sql.String(),
				})
				//TODO decode wheres
				it.DecodeWheres(wheres, mapper, logic, nil)
			}
			break
		} else {
			//default delete  DELETE FROM `test`.`biz_activity` WHERE `id`='165';
			var sql bytes.Buffer
			sql.WriteString("delete from ")
			sql.WriteString(tables)
			if len(wheres) > 0 {
				//sql.WriteString(" where ")
				mapper.Child = append(mapper.Child, &etree.CharData{
					Data: sql.String(),
				})
				//TODO decode wheres
				it.DecodeWheres(wheres, mapper, LogicDeleteData{}, nil)
			}
		}

	default:
		return false, nil
	}
	return true, nil
}

func checkTablesValue(mapper *etree.Element, tables *string, resultMapData *etree.Element) {
	if *tables == "" {
		*tables = resultMapData.SelectAttrValue("tables", "")
		if *tables == "" {
			panic("[GoMybatisTemplateDecoder] attribute 'tables' can not be empty! need define in <resultMap> or <" + mapper.Tag + "Template>,mapper id=" + mapper.SelectAttrValue("id", ""))
		}
	}
}

//解码逗号分隔的where
func (it *GoMybatisTemplateDecoder) DecodeWheres(arg string, mapper *etree.Element, logic LogicDeleteData, versionData *VersionData) {
	var whereRoot = &etree.Element{
		Tag:   Element_where,
		Attr:  []etree.Attr{},
		Child: []etree.Token{},
	}
	if logic.Enable == true {
		var appendAdd = ""
		var item = &etree.CharData{
			Data: appendAdd + logic.Column + " = " + logic.Undelete_value,
		}
		whereRoot.Child = append(whereRoot.Child, item)
	}
	if versionData != nil {
		var appendAdd = ""
		if len(whereRoot.Child) >= 1 {
			appendAdd = " and "
		}
		var item = &etree.CharData{
			Data: appendAdd + versionData.Column + " = #{" + versionData.Property + "}",
		}
		whereRoot.Child = append(whereRoot.Child, item)
	}

	var wheres = strings.Split(arg, ",")
	for index, v := range wheres {
		if v == "" || strings.Trim(v, " ") == "" {
			continue
		}
		var expressions = strings.Split(v, "?")
		var appendAdd = ""
		if index >= 1 || len(whereRoot.Child) > 0 {
			appendAdd = " and "
		}
		var item etree.Token
		if len(expressions) > 1 {
			//TODO have ?
			var newWheres bytes.Buffer
			newWheres.WriteString(expressions[1])

			item = &etree.Element{
				Tag:   Element_If,
				Attr:  []etree.Attr{{Key: "test", Value: it.makeIfNotNull(expressions[0])}},
				Child: []etree.Token{&etree.CharData{Data: appendAdd + newWheres.String()}},
			}
		} else {
			var newWheres bytes.Buffer
			newWheres.WriteString(appendAdd)
			newWheres.WriteString(v)
			item = &etree.CharData{
				Data: newWheres.String(),
			}
		}
		whereRoot.Child = append(whereRoot.Child, item)
	}
	mapper.Child = append(mapper.Child, whereRoot)
}

func (it *GoMybatisTemplateDecoder) DecodeSets(arg string, mapper *etree.Element, logic LogicDeleteData, versionData *VersionData) {
	var sets = strings.Split(arg, ",")
	for index, v := range sets {
		if v == "" {
			continue
		}
		var expressions = strings.Split(v, "?")
		if len(expressions) > 1 {
			var newWheres bytes.Buffer
			if index > 0 {
				newWheres.WriteString(",")
			}
			newWheres.WriteString(expressions[1])
			var item = &etree.Element{
				Tag:  Element_If,
				Attr: []etree.Attr{{Key: "test", Value: it.makeIfNotNull(expressions[0])}},
			}
			item.SetText(newWheres.String())
			mapper.Child = append(mapper.Child, item)
		} else {
			var newWheres bytes.Buffer
			if index > 0 {
				newWheres.WriteString(",")
			}
			newWheres.WriteString(v)
			var item = &etree.CharData{
				Data: newWheres.String(),
			}
			mapper.Child = append(mapper.Child, item)
		}
	}
	if logic.Enable == true {
		var appendAdd = ""
		if len(sets) >= 1 && arg != "" {
			appendAdd = ","
		}
		var item = &etree.CharData{
			Data: appendAdd + logic.Column + " = " + logic.Deleted_value,
		}
		mapper.Child = append(mapper.Child, item)
	}
	if versionData != nil {
		var appendAdd = ""
		if len(sets) >= 1 && arg != "" {
			appendAdd = ","
		}
		var item = &etree.CharData{
			Data: appendAdd + versionData.Column + " = #{" + versionData.Property + "+1}",
		}
		mapper.Child = append(mapper.Child, item)
	}
}

func (it *GoMybatisTemplateDecoder) makeIfNotNull(arg string) string {
	for _, v := range equalOperator {
		if v == "" {
			continue
		}
		if strings.Contains(arg, v) {
			return arg
		}
	}
	return arg + ` != nil`
}

func (it *GoMybatisTemplateDecoder) decodeLogicDelete(xml *etree.Element) LogicDeleteData {
	if xml == nil || len(xml.Child) == 0 {
		return LogicDeleteData{}
	}
	var logicData = LogicDeleteData{}
	for _, v := range xml.ChildElements() {
		if v.SelectAttrValue("logic_enable", "") == "true" {
			logicData.Enable = true
			logicData.Deleted_value = v.SelectAttrValue("logic_deleted", "")
			logicData.Undelete_value = v.SelectAttrValue("logic_undelete", "")
			logicData.Column = v.SelectAttrValue("column", "")
			logicData.Property = v.SelectAttrValue("column", "")
			logicData.LangType = v.SelectAttrValue("langType", "")
			//check
			if logicData.Deleted_value == "" {
				panic(utils.NewError("GoMybatisTemplateDecoder", `<resultMap> logic_deleted="" can't be empty !`))
			}
			if logicData.Undelete_value == "" {
				panic(utils.NewError("GoMybatisTemplateDecoder", `<resultMap> logic_undelete="" can't be empty !`))
			}
			if logicData.Undelete_value == logicData.Deleted_value {
				panic(utils.NewError("GoMybatisTemplateDecoder", `<resultMap> logic_deleted value can't be logic_undelete value!`))
			}
			break
		}
	}
	return logicData
}

func (it *GoMybatisTemplateDecoder) decodeVersionData(xml *etree.Element) *VersionData {
	if xml == nil || len(xml.Child) == 0 {
		return nil
	}
	for _, v := range xml.ChildElements() {
		if v.SelectAttrValue("version_enable", "") == "true" {

			var versionData = VersionData{}
			versionData.Column = v.SelectAttrValue("column", "")
			versionData.Property = v.SelectAttrValue("column", "")
			versionData.LangType = v.SelectAttrValue("langType", "")
			//check
			if !(strings.Contains(versionData.LangType, "int") || strings.Contains(versionData.LangType, "time.Time")) {
				panic(utils.NewError("GoMybatisTemplateDecoder", `version_enable only support int...,time.Time... number type!`))
			}
			return &versionData
		}
	}
	return nil
}

//反射解码得到 集合名词
func (it *GoMybatisTemplateDecoder) DecodeCollectionName(method *reflect.StructField) string {
	var collection string
	//check method arg type
	if method != nil {
		var numIn = method.Type.NumIn()
		for i := 0; i < numIn; i++ {
			var itemType = method.Type.In(i)
			if itemType.Kind() == reflect.Slice || itemType.Kind() == reflect.Array {
				var params = method.Tag.Get("args")
				var args = strings.Split(params, ",")
				if params == "" || args == nil || len(args) == 0 {
					collection = DefaultOneArg + strconv.Itoa(i)
				} else {
					if args[i] == "" {
						collection = DefaultOneArg + strconv.Itoa(i)
					} else {
						collection = args[i]
					}
				}
				if collection != "" {
					return collection
				}
			}
		}
	}
	return collection
}
