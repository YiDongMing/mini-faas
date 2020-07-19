# 实现一个 Serverless 计算服务调度系统

## 赛题背景

Serverless是最近几年一个很热门的技术话题，它的核心理念是让用户专注业务逻辑，把业务无关的事情（比如服务器管理等）交给云服务来做。函数即服务（Function as a Service，FaaS）是Serverless计算的一个重要组成部分，以[阿里云函数计算服务](https://www.aliyun.com/product/fc)为例，用户只需要用函数实现业务逻辑，上传函数代码，函数计算服务会准备好计算资源，并以弹性、可靠的方式运行用户代码，支撑了像新浪微博的图片处理等[多种场景](https://resources.functioncompute.com/solutions.html)。本题目从比赛的角度对 FaaS 服务进行了一些抽象和简化，让选手了解FaaS类云服务背后的架构和解决其中一些有意思的问题。需要说明的是，本赛题的问题和解读不代表阿里云函数计算的实现。

函数计算服务跟其他一些计算服务的最大区别是，在函数计算中计算资源的所有权是服务方，由服务负责资源的利用率。这样对用户来说它的资源利用率就是100%，函数执行了100ms，那么只会收取100ms的费用，没有资源闲置和浪费，这就是所谓的**按使用付费**。具体的说，按照函数的实际执行时间和运行函数容器的规格收费。一个调用请求会触发函数执行，执行结束后不再收取费用。如果使用传统的服务器运行应用，那不管应用是否有访问，使用服务器都会产生费用。

## 赛题描述

### 术语定义

1. FaaS是Serverless计算服务的一种形态但不是唯一形态，阿里云函数计算是FaaS的一个具体实现。在本赛题中可能会交替使用。
2. Node是一个抽象的概念，是函数运行环境的载体，其具体实现可以是虚拟机或物理机，在本赛题中使用阿里云ECS虚拟机。
3. 容器（Container）是一个抽象的概念，代表的是函数的运行环境（Runtime），一个创建好的容器只能执行特定的函数。在本赛题中使用Docker Container。

### 问题描述

FaaS服务方是如何提供服务的呢？其中一个核心的模块是调度器，调度要回答的问题是函数调用的请求分配到哪个容器实例上执行。首先服务不可能为每个函数准备一个或者多个一直运行的服务器（或者容器等计算资源），这样会大大增加服务方的成本。一个最简单解决思路是等请求到来时再准备计算资源，执行结束后释放资源。这种方式的一个问题是准备函数执行的环境（Runtime）通常需要花费一定的时间，即所谓的冷启动，会增加调用的响应时间。类似的，如果执行完立刻释放资源，可能下一个请求会很快来到，再次准备环境又造成冷启动，拉长响应时间。**一个好的调度器是让调用请求能够被及时处理，又使用较少的资源**，这也是评测的标准。

一个基本的做法是：
1. 每个函数会维护一个容器集合，初始集合为空
2. 当函数调用请求达到时，从集合中选择一个Idle容器来执行请求，选中的容器被标记为Busy状态
3. 当集合中没有Idle容器可用时，就创建新的容器加入到集合中
4. 当函数调用执行完成时，就把容器标记为Idle状态
5. 当Idle容器闲置一段时间内没有请求时，就释放掉

这样做的好处有：
1. 能够精确地把请求分配到容器上执行，不会overload容器造成排队延时
2. 能够精确地把空闲容器释放，不会浪费容器资源
3. 能够快速地响应负载变化，弹性更加平滑

容器和机器还存在多对一的关系，一个机器按内存划分，可以创建多个容器。比赛使用的Node规格是[ecs.c6.large]((https://help.aliyun.com/document_detail/108491.html#section-0yl-3wv-ims))，2C4G（2核4GB内存）ECS虚拟机，容器的最大内存由函数指定（函数内存），CPU和内存按比例分配。

在上述的调度模型下，主要的问题可以分为两类：
* Scaling问题：如何保持合适的容器数来匹配函数的负载变化
  * 初次启动容器的冷启动会增加响应时间，容器闲置多长时间才释放？如何预测函数的负载变化，主动进行调度？
  * 当某个瞬间多个请求到来时，没有Idle容器，是否需要为每个请求创建容器？
* Placement问题：如何把容器放置到合适的Node上来减少资源浪费
  * Scheduler可能创建了多个Node，当请求到来时选择哪个Node创建容器？这时候一般要考虑尽量将容器紧密的放置在Node上，提高容器密度，减少Node的使用个数；同时又要考虑不同容器的影响，多个CPU instensive的负载会互相影响。
  * 当有多个容器Idle时，选择哪个容器来执行函数？
  * 如何定义容器Busy？最保守的做法是一个容器只处理一个并发请求，但是对于有些类型的函数，处理多个请求对执行时间影响不大。
  * 一个Node可以创建多少容器？因为容器是根据函数内存指定了最大内存限制，保守的做法是按照最大内存决定一个Node的容器个数，但是对于有些类型的函数，同一个Node创建更多的容器是可行的。

### 赛题设置

一个简化的FaaS系统分为APIServer，Scheduler，ResourceManager，NodeService，ContainerService 5个组件，本题目中APIServer，ResourceManager，NodeService，ContainerService由平台提供，Scheduler的```AcquireContainer```和```ReturnContainer```API由选手实现（gRPC服务，语言不限），在本赛题中Scheduler无需考虑分布式多实例问题，Scheduler以容器方式单实例运行。

![arch](https://cdn.nlark.com/yuque/0/2020/png/597042/1594746644546-abb9fa4e-e785-4d0c-9b60-a2049e66683b.png)

由上图我们可以看到，Scheduler会调用ResourceManager和NodeService的API，被APIServer调用，图中连线描述表示可以调用的API。

其中测试函数由平台提供，可能包含但不局限于helloworld，CPU intensive，内存intensive，sleep等类型；调用模式包括稀疏调用，密集调用，周期调用等；执行时间包括时长基本固定，和因输入而异等。

选手对函数的实现无感知，可以通过Scheduler的`AcquireContainer`和`ReturnContainer` API的调用情况，以及`NodeService.GetStats` API获得一些信息，用于设计和实现调度策略。
1. 通过`AcquireContainer`的调用反映了InvokeFunction调用的开始时间。
2. `ReturnContainer`的调用包含了请求级别的函数执行时间。
3. `NodeService.GetStats`返回了Container级别的资源使用情况和函数执行时间统计。


下面是各个组件的介绍：

**APIServer**：评测程序调用`APIServer.InvokeFunction`执行函数。

  * ListFunctions：返回所有可调用Function信息。本题目会预先定义一些Function，选手无需自行创建Function。
  * InvokeFunction(Function，Event)：执行某个Function，传入Event。

**Scheduler**：Scheduler管理系统的Container。APIServer通过Scheduler获得可以执行Function的Container。

  * AcquireContainer(Function)：获得一个可以执行Function的Container。如果系统有Container可以运行Function，Scheduler可以直接返回Container，否则Scheduler需要通过ResourceManager获得Node，进而在Node上创建Container。
  * ReturnContainer(Container)：归还Container。APIServer在调用`NodeService.InvokeFunction`后会调用ReturnContainer归还Container。Scheduler可以利用`AcquireContainer`和`ReturnContainer`记录Container的使用情况。

**ResourceManage**r：ResourceManager管理系统里的Node，负责申请和释放。可以认为Node对应于虚拟机，占用虚拟机需要一定的成本，因此Scheduler的一个目标是如何最大化的利用Node，比如尽量创建足够多的Container，不用的Node应该尽快释放。当然申请和释放Node又需要一定的时间，造成延迟增加，Scheduler需要平衡延迟和资源使用时间。

  * ReserveNode：申请一个Node，该API返回一个Node地址，使用该地址可以创建Container或者销毁Container。
  * ReleaseNode：释放Node。

**NodeService**：NodeService管理单个Node上创建和销毁Container，Node可以认为是一个虚拟机，Scheduler可以在Node上创建用于执行Function的Container。

  * CreateContainer(Function)：创建Container，该API返回Node地址和Container ID，使用改地址可以InvokeFunction。Scheduler可以通过`Node.DestroyContainer`销毁Container。
  * RemoveContainer(Container)：销毁Container。
  * InvokeFunction(Container, Event)：执行Node上的某个Container所加载的函数。
  * GetStats()：返回该Node下所有Container和执行环境相关指标，比如CPU，内存。

**ContainerService**：ContainerService用于执行函数。**注意，一个为某个Function创建的Container不能用来执行其它函数，但是可以串行或者并行执行同一个函数**。Scheduler不直接与ContainerService交互。

  * InvokeFunction(Event)：执行函数。

下面的时序图描述了3种常见场景下各组件调用关系：

![faas-sequence](https://cdn.nlark.com/yuque/0/2020/png/597042/1594747370103-03556b1e-c73a-4fe5-9205-e048dd7c200b.png) 

1. 当Client发起`InvokeFunction`请求时，APIServer调用Scheduler获得执行函数所需要的Container，Scheduler发现没有Node可用，因此调用ResourceManager预留Node，然后使用NodeService创建Container，返回Node地址和Container信息给APIServer。APIServer通过`NodeService.InvokeFunction`执行函数，返回结果给Client端，APIServer通过ReturnContainer告诉Scheduler该Container使用完毕，Scheduler可以决定是释放该Container或者将该Container处理下一个请求。注意：每个Container只能执行同一个函数。
2. 当Client调用同一个函数时，APIServer调用Scheduler获得执行函数所需要的Container，Scheduler发现已经为该函数创建过Container，所以直接将Node地址和Container信息返回给APIServer，之后的调用步骤与上面的场景一致。注意：如果APIServer在尚未将Container归还给Scheduler时，Scheduler是否可以将该Container返回给APIServer呢？一般来说，这取决于这个Container是否有资源来处理多个请求。这是Scheduler需要探索的地方之一。
3. 当Client调用另一个函数时，APIServer调用Scheduler获得执行函数所需要的Container，Scheduler发现该函数没有可用的Container，但是Node上还可以创建更多的Container，因此它调用NodeService为这个函数创建新的Container，并返回Node地址和Container信息给APIServer，之后的调用步骤与上面的场景一致。

**说明**：

1. Node的规格是[ecs.c6.large](https://help.aliyun.com/document_detail/108491.html#section-0yl-3wv-ims)，2C4G（2核4GB内存）ECS虚拟机，每小时0.39元，按秒计费。线上评测有最多20台Node可用，持续时间20分钟。线下测试需要自行提供ECS虚拟机，线下评测平台会根据选手提供的阿里云AccessKey创建ECS虚拟机，选手可以配置虚拟机个数，进行测试，比如20台机器运行20分钟的花费是2.6元（20x20x0.39/60）。测试完成后确保释放所有的ECS虚拟机，以免产生额外费用。
2. Function运行所需要的容器规格由Function Meta的`memory_in_bytes`决定，在```Scheduler.AcquireContainer```时通过```memory_in_bytes```参数传入，Scheduler在调用```NodeService.CreateContainer```传入该参数，NodeService会根据该参数作为[Memory的最大限制](https://docs.docker.com/config/containers/resource_constraints/)创建Container加载函数。选手可以选择在一个Node上创建多于4GB内存的Container（超卖），但在某些情况下可能会影响性能甚至导致Function执行OOM（Out Of Memory）。
3. Scheduler运行在4C8G ECS虚拟机上的容器内，无公网访问能力。
4. Scheduler需要调用ResourceManager的地址通过环境变量（`RESOURCE_MANAGER_ENDPOINT`）获得，端口是`10400`。NodeService的访问地址可以从`ResourceManager.ReserveNode`的[返回](https://code.aliyun.com/middleware-contest-2020/mini-faas/blob/master/scheduler/core/router.go#L95-96)中获得。
5. Scheduler gRPC Server监听端口是`10600`，如[示例](https://code.aliyun.com/middleware-contest-2020/mini-faas/blob/master/scheduler/main.go#L103)所示。
6. 日志写到stdout，在评测结果中提供，有效期1天。

## 评测方式和标准

**指标定义**

* 资源使用时间（ND：Node Duration）：Node使用时间。获得Node成功后开始计时，同一Node释放成功后结束计时，其差值为资源使用时间。
* 调度延迟（SL: Schedule Latency）：调用Scheduler.AcquireContainer所需时间。
* 函数执行时间（FD：Function execution Duration）：调用ContainerService.InvokeFunction所需时间。
* 响应时间（RT：Response Time）：调度延迟 + 函数执行时间。
* 请求成功率：调用APIServer.InvokeFunction成功的次数与APIServer.InvokeFunction调用总次数的比值。

评测程序会对函数以不同QPS（Query Per Second）调用APIServer.InvokeFunction。比如n个函数，f1调用c1次，f2调用c2次，...，fn调用cn次，系统记录总的资源使用时间，每次调用的响应时间，以及每次调用是否成功。**调度的目标是在满足调用成功率的前提下，平衡总的响应时间（RT）和资源使用时间（ND）**。


**如何评测**

1. 评测程序会首先根据基准Scheduler对评测数据计算出基准响应时间（RT_f1b，RT_f2b，...，RT_fnb）和资源使用时间（ND_b）。其中RT_fib是对函数i的所有调用的响应之和。
2. 评测程序会对选手实现的Scheduler测试同样的数据，计算响应时间（RT_f1，RT_f2，...，RT_fn），资源使用时间（ND）和请求成功率。其中RT_fi是对函数i的所有调用的响应之和。
3. Score_ti = (RT_f1/RT_f1b + RT_f2/RT_f2b + ... + RT_fn/RT_fnb)/n x Weight1 + (ND / NDb) x Weight2。其中权重Weight1和Weight2暂不设定（后续公布），Scheduler的调度策略需要将权重作为调度的一个配置，比如Weight1=0.5，Weight2=0.5，或者Weight1=0.75，Weight2=0.25。


分数计算示例

| 指标             | 基准得分 | 全优选手  | 均衡选手 | 响应优选手 | 资源低选手 |
| ---------------- | -------- | --------- | -------- | ---------- | ---------- |
| RT_f1            | 5        | 4         | 6        | 4          | 7          |
| RT_f2            | 10       | 10        | 8        | 8          | 12         |
| RT_f3            | 8        | 7         | 10       | 5          | 10         |
| RT_f4            | 10       | 8         | 10       | 8          | 12         |
| ND               | 40       | 35        | 45       | 60         | 30         |
| Score(0.5+0.5)   | 1        | 0.871875  | 1.09375  | 1.153125   | 1.00625    |
| Score(0.75+0.25) | 1        | 0.8703125 | 1.078125 | 0.9421875  | 1.134375   |



## 开发环境搭建

## 本地测试

本赛题会提供本地测试环境，本地测试环境跟线上评测环境相比，用到的测试函数和测试数据不同，但是又有一定的代表性，比如其中测试函数可能包含但不局限于helloworld，CPU intensive，内存intensive，sleep等类型；调用模式包括稀疏调用，密集调用，周期调用等；执行时间包括时长基本固定，和因输入而异等。**本地测试以验证功能为主**。



```
mkdir -p $GOPATH/src/aliyun/serverless
cd $GOPATH/src/aliyun/serverless
git clone https://code.aliyun.com/middleware-contest-2020/mini-faas.git
cd mini-faas
```



TODO：

1. <strike>Java语言不会提供完整示例，后续（7月18日前）会提供基本接口实现和调用示例。</strike>
2. 后续（7月18日前）会提供本地联调。



## 提交线上测试

为保证评测程序正常运行出分，请提交阿里云ACR镜像，仓库地域选择杭州。具体参考：https://tianchi.aliyun.com/competition/entrance/231759/tab/174

## 参考

1. [阿里云函数计算服务](https://www.aliyun.com/product/fc)
2. [Serverless in the Wild: Characterizing and Optimizing the Serverless Workload at a Large Cloud Provider](https://arxiv.org/pdf/2003.03423.pdf) 和[解读](https://mikhail.io/2020/06/eliminate-cold-starts-by-predicting-invocations-of-serverless-functions/)
3. [OpenFaaS自动伸缩介绍](https://docs.openfaas.com/architecture/autoscaling/) 
4. [Knative自动伸缩介绍](https://knative.dev/docs/serving/configuring-autoscaling/) 