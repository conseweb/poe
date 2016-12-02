# poe
Proof of Exists

## 开发测试流程
* 添加了`vendor`目录，以后的修改需要在本地执行`make test` 通过后提交pr，有需要引用其它第三方包的需要通过`govendor` 命令进行添加或者版本的更新。
* 添加了`make image`命令，构建运行时的镜像.
使用：
```
docker run --rm --name poe \
 -e POE_CACHE_KAFKA_BROKERS="192.168.5.101:9092 192.168.5.102:9092 192.168.5.103:9092" \
 -e POE_PERSIST_CASSANDRA_CLUSTERS="u2.mj u3.mj u4.mj"  \
conseweb/poe:(本地git branch)
```
* 使用`make test`进行单元测试

**注:本地测试通过后再提交pr**

## API 接口
平台暂定三个接口,如下:

### POST /api/v1/documents
新增一个需要证明存在性的文档

#### 路径参数
无

#### 查询参数
无

#### JSONBODY
* proofWaitPeriod: 期待文件证明完成所需要的时间,期待时间越短,相对应的API请求次数越少,或者请求代价越高。可选参数有 1m、10m、30m、1h、3h、12h、24h
* rawDocument: 需要被证明的文件原始内容(字符串类型),当然如果不想传输机密文件内容,可先对数据进行加密混淆,相对应的在验证文件是否存在的时候也需要加密混淆

#### 返回状态值
201 400 500

#### 返回值(json)
* documentId: 文件对应ID,此ID仅用于平台在证明过程中查询文件的证明状态,不可作为文件存在的凭证
* perdictProofTime: 预计文件能被证明的UTC时间戳

### GET /api/v1/documents/:id/status
查询文件存在性证明过程

#### 路径参数
* id: 上一个API返回的参数,标识在存在性证明过程中的一个文件

#### 查询参数
无

#### Form表单
无

#### 返回状态值
200 404

#### 返回值(json)
* status: 文件证明状态
    * none: 没有该ID的存在证明记录,此时不意味着之前的新增无效,可能间隔时间太短,系统还未处理
    * wait: 系统已经在处理文件证明了,只是还没有处理完成
    * ok: 文件已经进行了存在性证明
* documentId: 输入的路径参数,直接返回
* documentBlockDigest: 文件证明所返回的摘要信息
* perdictProofTime: 如果状态是等待中,则显示预计的证明时间
* proofTime: 如果状态是已证明,则显示证明时间

### POST /api/v1/documents/result
验证文件的存在性

#### 路径参数
无

#### 查询参数
无

#### JSONBODY
* rawDocument: 需要被证明的文件原始内容(字符串类型),当然如果不想传输机密文件内容,可先对数据进行加密混淆,相对应的在验证文件是否存在的时候也需要加密混淆

#### 返回状态值
200 404

#### 返回值(json)
* status: 验证结果
    * invalid: 未被证明
    * wait: 等待被证明
    * valid: 已被证明
    * none: 无此结果
* documentId: 文件证明时分配的ID
* submitTime: 文件提交时间(UTC 时间戳)
* proofTime: 文件被证明时间(UTC 时间戳)
* perdictProofTime: 当状态为等待被证明时,预计被证明的时间(UTC 时间戳)

### GET /api/v1/documents
获取最近n条文件

#### 路径参数
无

#### 查询参数
* count: 需要查询的个数
* type: 查询的类型, register(已注册未证明)、proof(已证明)

#### Form表单
无

#### 返回状态值
200 400 500

#### 返回值(json)
* docs: 查询的文件列表



## 模块架构

### api
对外接口

### blockchain
文件存在性证明,连接hyperledger fabric chaincode

### cache
缓存api请求

### persist
从cache获取documents进行持久化

