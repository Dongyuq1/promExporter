# 自定义Prometheus Exporter拉取MongoDB内的数据

## 实现Exporter部分

很多时候，我们在使用Prometheus时，官方提供的采集组件不能满足监控需求，我们就需要自行编写Exporter。

本文的示例采用go语言和Gauge (测量指标)类型实现。自定义Exporter去取MongoDB里动态增长的数据。

#### Metric接口

Prometheus client库提供了四种度量标准类型。 

虽然只有基本度量标准类型实现Metric接口，但是度量标准及其向量版本都实现了Collector接口。Collector管理许多度量标准的收集，但为方便起见，度量标准也可以"自行收集"。Prometheus数据模型的一个非常重要的部分是沿称为label的维度对样本进行划分，从而产生度量向量。基本类型是GaugeVec，CounterVec，SummaryVec和HistogramVec。注意，Gauge，Counter，Summary和Histogram本身是接口，而GaugeVec，CounterVec，SummaryVec和HistogramVec则不是接口。

- Counter (累积)

  Counter一般表示一个单调递增的计数器。

- Gauge (测量)

  Gauge一般用于表示可能动态变化的单个值，可能增大也可能减小。

- Histogram (直方图)

- Summary (概略图)

#### 注册指标并启动HTTP服务

需要先引入这个Prometheus client库。通常metric endpoint使用http来暴露metric，通过http暴露metric的工具为promhttp子包，引入promhttp和net/http。

```Golang
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang//prometheus/promhttp"
```

Registry/Register/Gatherers及http服务

```Golang
func main() {
	daystr := time.Now().Format("20060102")
	logFile, err := os.Create("./exporter/log/" + daystr + ".txt")
	defer logFile.Close()
	if err != nil {
		fmt.Printf("%v\n", err)
		return
	}
	logger := log.New(logFile, "Prefix_", log.Ldate|log.Ltime|log.Lshortfile)
	
	//Registry和Register部分
	reg := prometheus.NewPedanticRegistry()
	reg.MustRegister(metrics.initCollector())    //metrics.initCollector()是collector初始化模块
	//以下是官方文档定义
	// Register registers a new Collector to be included in metrics
	// collection. It returns an error if the descriptors provided by the
	// Collector are invalid or if they — in combination with descriptors of
	// already registered Collectors — do not fulfill the consistency and
	// uniqueness criteria described in the documentation of metric.Desc.
	//
	// If the provided Collector is equal to a Collector already registered
	// (which includes the case of re-registering the same Collector), the
	// returned error is an instance of AlreadyRegisteredError, which
	// contains the previously registered Collector.
	//
	// A Collector whose Describe method does not yield any Desc is treated
	// as unchecked. Registration will always succeed. No check for
	// re-registering (see previous paragraph) is performed. Thus, the
	// caller is responsible for not double-registering the same unchecked
	// Collector, and for providing a Collector that will not cause
	// inconsistent metrics on collection. (This would lead to scrape
	// errors.)

	// MustRegister works like Register but registers any number of
	// Collectors and panics upon the first registration that causes an
	// error.

	//Gatherers部分
	gatherers := prometheus.Gatherers{
		reg,
	}
	//以下是官方文档定义
	// Gatherers is a slice of Gatherer instances that 
	// implements the Gatherer interface itself.
	// Its Gather method calls Gather on all Gatherers 
	// in the slice in order and returns the merged results.
	// Errors returned from the Gather calls are 
	// all returned in a flattened MultiError.
	// Duplicate and inconsistent Metrics are 
	// skipped (first occurrence in slice order wins) 
	// and reported in the returned error.

	//注册http服务
	h := promhttp.HandlerFor(gatherers,
		promhttp.HandlerOpts{
			ErrorHandling: promhttp.ContinueOnError,
		})
	http.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		h.ServeHTTP(w, r)
	})
	log.Println("Start server at :8710")
	logger.Printf("Start server at :8710")

	if err := http.ListenAndServe(":8710", nil); err != nil {
		log.Printf("Error occur when start server %v", err)
		logger.Printf("Error occur when start server %v", err)
		os.Exit(1)
	}		
}	
```

Registry是为了根据Prometheus数据模型来确保所收集指标的一致性。如果注册的Collector与注册的Metrics不兼容或不一致，则返回错误。理想情况下，在注册时而不是在收集时检测到不一致。对注册的Collector的检测通常会在程序启动时被检测到，而对注册的Metrics的检测只会在抓取时发生，这就是Collector和Metrics必须向Registry描述自己的主要原因。

Registry实现了Gatherer接口。 然后Gather方法的调用者可以某种方式公开收集的指标。通常度量是通过/metrics端点上的HTTP提供的。在上面的示例中就是这种情况。通过HTTP公开指标的工具位于promhttp子软件包中。

NewPedanticRegistry可以避免由DefaultRegisterer施加的全局状态，可以同时使用多个注册表，以不同的方式公开不同的指标， 也可以将单独的注册表用于测试目的。

#### 实现Collector接口

```Golang
type Collector interface {
    // 用于传递所有可能的指标的定义描述符
    // 可以在程序运行期间添加新的描述，收集新的指标信息
    // 重复的描述符将被忽略。两个不同的Collector不要设置相同的描述符
    Describe(chan<- *Desc)

    // Prometheus的注册器调用Collect执行实际的抓取参数的工作，
    // 并将收集的数据传递到Channel中返回
    // 收集的指标信息来自于Describe中传递，可以并发的执行抓取工作，但是必须要保证线程的安全。
    Collect(chan<- Metric)
}
```

在另外的模块中编写Collector以及其初始化的代码。

先定义一个结构体，指标使用的是 prometheus的Desc类型。

```Golang
type ProjectMetrics struct {
	MetricsDescs []*prometheus.Desc
}
```

定义MetricsName和 MetricsHelp。

```
var MetricsNameXXX = "XXX"
var MetricsHelpXXX = "(XXX)"
```

对自定义类型ProjectMetrics定义Describe方法和Collect方法来实现实现Collector接口。

```Golang
func (c *ProjectMetrics) Describe(ch chan<- *prometheus.Desc) {
	len1 := len(c.MetricsDescs)
	for i := 0; i < len1; i++ {
		ch <- c.MetricsDescs[i]
	}
}

func (c *ProjectMetrics) Collect(ch chan<- prometheus.Metric) {
	start := time.Now()
	nowUTC := start.UTC()
	resp := mongodb.QueryAllData(nowUTC.Unix())
	for _, v := range resp.Data1Tier {
		item1 := v.Item1
		fmt.Println("......................", isp)
		ts := time.Unix(v.Clientutc, 0)
		for _, v2 := range v.Data2Tier {
			item2 := v2.Item2
			item3 := v2.Item3
			item4 := v2.Item4
			tmp := prometheus.NewDesc(
				MetricsNameXXX,
				MetricsHelpXXX,
				[]string{"Name"},
				prometheus.Labels{"item1": item1, "item3": item3, "item4": item4},
			)
			ch <- prometheus.NewMetricWithTimestamp(
				ts,
				prometheus.MustNewConstMetric(
					tmp,
					prometheus.GaugeValue,
					item2,
					MetricsNameXXX,
				),
			)
		}
	}
	eT := time.Since(start)
	fmt.Printf("Project Metrics, Elapsed Time: %s, Date(UTC): %s\n", eT, start.UTC().Format("2006/01/02T15:04:05"))
}
```

初始化ProjectMetrics，

```
func AddMetricsItem2() *ProjectMetrics {
	var tmpMetricsDescs []*prometheus.Desc
	resp := mongodb.QueryAllData(time.Now().UTC().Unix())
	for _, v := range resp.Data1Tier {
		item1 := v.Item1
		for _, v2 := range v.Data2Tier {

			item3 := v2.Item3
			item4 := v2.Item4
			tmp := prometheus.NewDesc(
				MetricsNameLatency,
				MetricsHelpLatency,
				[]string{"Name"},
				prometheus.Labels{"item1": item1, "item3": item3, "item4": item4},
			)
			tmpMetricsDescs = append(tmpMetricsDescs, tmp)
		}
	} //aws

	api := &ProjectMetrics{MetricsDescs: tmpMetricsDescs}
	return api
}
```

MongoDB中的数据，

```Golang
type SensorData struct {
	Aaaa         string      `json:"aaaa"`
	Item2        float64     `json:"item2"`
	Item4            string      `json:"item4"`
	Item3 string      `json:"item3"`
	Bbbb          float64     `json:"bbbb"`
	Cccc         string      `json:"cccc"`
	Dddd         []CmdResult `json:"dddd"`
	Eeee           string      `json:"eeee"`
}

type Sensor struct {
	Item1         string `json:"item1"`
	Clientutc   int64  `json:"clientutc"`
	Data2Tier []SensorData
}

type AllData struct {
	Data1Tier []Sensor
}
```

mongodb.QueryAllData()是另外从mongodb中拉取数据的模块，返回AllData类型数据。mongodb模块每三分钟去数据库里拉一次数据。

## 实现MongoDB部分

### MongoDB的Go驱动包

```
"go.mongodb.org/mongo-driver/bson"    //BOSN解析包
"go.mongodb.org/mongo-driver/mongo"    //MongoDB的Go驱动包
"go.mongodb.org/mongo-driver/mongo/options"
```

###### BSON简介

BSON是一种类json的一种二进制形式的存储格式，简称Binary JSON。MongoDB使用了BSON这种结构来存储数据和网络数据交换。

BSON对应`Document`这个概念，因为BSON是schema-free的，所以在MongoDB中所对应的`Document`也有这个特征，这里的一个`Document`也可以理解成关系数据库中的一条`Record`，只是`Document`的变化更丰富一些，`Document`可以嵌套。

MongoDB以BSON做为其存储结构的一个重要原因是它的`可遍历性`。

BSON编码扩展了JSON表示，使其包含额外的类型，如int、long、date、浮点数和decimal128。

###### BSON类型

BSON数据的主要类型有：`A`，`D`，`E`，`M`和`Raw`。其中，`A`是数组，`D`是切片，`M`是映射，`D`和`M`是Go原生类型。

- `A`类型表示有序的BSON数组。

  ```
  bson.A{"bar", "world", 3.14159, bson.D{{"qux", 12345}}}
  ```

- `D`类型表示包含有序元素的BSON文档。这种类型应该在顺序重要的情况下使用。如果元素的顺序无关紧要，则应使用M代替。

  ```
  bson.D{{"foo", "bar"}, {"hello", "world"}, {"pi", 3.14159}}
  ```

- `M`类型表示无序的映射。

  ```
  bson.M{"foo": "bar", "hello": "world", "pi": 3.14159}
  ```

- `E`类型表示D里面的一个BSON元素。

- `Raw`类型代表未处理的原始BSON文档和元素，`Raw`系列类型用于验证和检索字节切片中的元素。当要查找BSON字节而不将其解编为另一种类型时，此类型最有用。

### 连接到mongoDB

```
// 设置mongoDB客户端连接信息
param := fmt.Sprintf("mongodb://XXX.XXX.XXX.XXX:27017")
clientOptions := options.Client().ApplyURI(param)

// 建立客户端连接
client, err := mongo.Connect(context.TODO(), clientOptions)
if err != nil {
log.Fatal(err)
fmt.Println(err)
}

// 检查连接情况
err = client.Ping(context.TODO(), nil)
if err != nil {
log.Fatal(err)
fmt.Println(err)
}
fmt.Println("Connected to MongoDB!")

//指定要操作的数据集
collection := client.Database("ccmsensor").Collection("mtr")

//执行增删改查操作

// 断开客户端连接
err = client.Disconnect(context.TODO())
if err != nil {
log.Fatal(err)
}
fmt.Println("Connection to MongoDB closed.")
```

### 增查改删

假如数据库中有一些网络连接数据，来自不同的APP，来自不同的ISP（运营商），类型如下：

```Golang
type CurlInfo struct {
	DNS float64 `json:"NAMELOOKUP_TIME"` //NAMELOOKUP_TIME
	TCP float64 `json:"CONNECT_TIME"`    //CONNECT_TIME - DNS
	SSL float64 `json:"APPCONNECT_TIME"` //APPCONNECT_TIME - CONNECT_TIME
}

type ConnectData struct {
	Latency  float64  `json:"latency"`
	RespCode int      `json:"respCode"`
	Url      string   `json:"url"`
	Detail   CurlInfo `json:"details"`
}

type Sensor struct {
	ISP       string
	Clientutc int64
	DataByAPP map[string]ConnectData
}
```

##### 增加

使用`collection.InsertOne()`来插入一条`Document`记录：

```Golang
func insertSensor(client *mongo.Client, collection *mongo.Collection) (insertID primitive.ObjectID) {
	apps := make(map[string]ConnectData, 0)
	apps["app1"] = ConnectData{
		Latency:  30.983999967575,
		RespCode: 200,
		Url:      "",
		Detail: CurlInfo{
			DNS: 5.983999967575,
			TCP: 10.983999967575,
			SSL: 15.983999967575,
		},
	}
	
	record := &Sensor{
		Clientutc: time.Now().UTC().Unix(),
		ISP:       "China Mobile",
		DataByAPP: apps,
	}

	insertRest, err := collection.InsertOne(context.TODO(), record)
	if err != nil {
		fmt.Println(err)
		return
	}
	insertID = insertRest.InsertedID.(primitive.ObjectID)
	return insertID
}
```

##### 查询

这里引入一个`filter`来匹配MongoDB数据库中的`Document`记录，使用`bson.D`类型来构建`filter`。

```Golang
timestamp := time.Now().UTC().Unix()
start := timestamp - 180
end := timestamp

filter := bson.D{
{"isp", isp},
{"$and", bson.A{
bson.D{{"clientutc", bson.M{"$gte": start}}},
bson.D{{"clientutc", bson.M{"$lte": end}}},
}},
}
```

使用`collection.FindOne()`来查询单个`Document`记录。这个方法返回一个可以解码为值的结果。

```Golang
func querySensor(collection *mongo.Collection, isp string) {
	//查询一条记录

	//筛选数据
	timestamp := time.Now().UTC().Unix()
	start := timestamp - 1800
	end := timestamp

	filter := bson.D{
		{"isp", isp},
		{"$and", bson.A{
			bson.D{{"clientutc", bson.M{"$gte": start}}},
			bson.D{{"clientutc", bson.M{"$lte": end}}},
		}},
	}

	var original Sensor
	err := collection.FindOne(context.TODO(), filter).Decode(&original)
	if err != nil {
		fmt.Printf("%s\n", err.Error())
	}
	fmt.Printf("Found a single document: %+v\n", original)
}
```

结果为刚刚插入的那一条数据，

```Golang
Connected to MongoDB!
Found a single document: {ISP:China Mobile Clientutc:1598867346 DataByAPP:map[app1:{Latency:30.983999967575 RespCode:200 Url: Detail:{DNS:5.983999967575 TCP:10.983999967575 SSL:15.983999967575}}]}
Connection to MongoDB closed.
```

若要提取其中的数据，在querySensor(）方法中略作改动，

```
original := make(map[string]interface{})
var sensorData Sensor
err := collection.FindOne(context.TODO(), filter).Decode(&original)
if err != nil {
	fmt.Printf("%s\n", err.Error())
} else {
	vstr, okstr := original["isp"].(string)
	if okstr {
	sensorData.ISP = vstr
	}
}
```

##### 更新

这里仍然使用刚才的filter，并额外需要一个`update`。

```
update := bson.M{
"$set": bson.M{
"isp": ispAfter,
},
}
```

使用`collection.UpdateOne()`更新单个`Document`记录。

```
func UpdateSensor(collection *mongo.Collection, ispBefore string, ispAfter string) {
	//修改一条数据

	//筛选数据
	timestamp := time.Now().UTC().Unix()
	start := timestamp - 1800
	end := timestamp

	filter := bson.D{
		{"isp", ispBefore},
		{"$and", bson.A{
			bson.D{{"clientutc", bson.M{"$gte": start}}},
			bson.D{{"clientutc", bson.M{"$lte": end}}},
		}},
	}

	//更新内容
	update := bson.M{
		"$set": bson.M{
			"isp": ispAfter,
		},
	}

	updateResult, err := collection.UpdateOne(context.TODO(), filter, update)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Matched %v documents and updated %v documents.\n", updateResult.MatchedCount, updateResult.ModifiedCount)
}
```

##### 删除

使用`collection.DeleteOne()`来删除一条记录，仍然使用刚才的filter。

```
deleteResult, err := collection.DeleteOne(context.TODO(), filter)
if err != nil {
	fmt.Printf("%s\n", err.Error())
}
```

更多操作请见官方文档。

###### 参考链接

[Mongo-Driver驱动包官方文档](https://pkg.go.dev/go.mongodb.org/mongo-driver@v1.4.0)
[BSON包官方文档](https://pkg.go.dev/go.mongodb.org/mongo-driver@v1.4.0/bson?tab=doc)
[mongo包官方文档](https://pkg.go.dev/go.mongodb.org/mongo-driver@v1.4.0/mongo?tab=doc)
[options包官方文档](https://pkg.go.dev/go.mongodb.org/mongo-driver@v1.4.0/mongo/options?tab=doc)

