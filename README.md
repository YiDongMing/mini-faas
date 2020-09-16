# 实现一个 Serverless 计算服务调度系统

## 赛题背景

Serverless是最近几年一个很热门的技术话题，它的核心理念是让用户专注业务逻辑，把业务无关的事情（比如服务器管理等）交给云服务来做。函数即服务（Function as a Service，FaaS）是Serverless计算的一个重要组成部分，以[阿里云函数计算服务](https://www.aliyun.com/product/fc)为例，用户只需要用函数实现业务逻辑，上传函数代码，函数计算服务会准备好计算资源，并以弹性、可靠的方式运行用户代码，支撑了像新浪微博的图片处理等[多种场景](https://resources.functioncompute.com/solutions.html)。本题目从比赛的角度对 FaaS 服务进行了一些抽象和简化，让选手了解FaaS类云服务背后的架构和解决其中一些有意思的问题。需要说明的是，本赛题的问题和解读不代表阿里云函数计算的实现。

函数计算服务跟其他一些计算服务的最大区别是，在函数计算中计算资源的所有权是服务方，由服务负责资源的利用率。这样对用户来说它的资源利用率就是100%，函数执行了100ms，那么只会收取100ms的费用，没有资源闲置和浪费，这就是所谓的**按使用付费**。具体的说，按照函数的实际执行时间和运行函数容器的规格收费。一个调用请求会触发函数执行，执行结束后不再收取费用。如果使用传统的服务器运行应用，那不管应用是否有访问，使用服务器都会产生费用。

本项目为参加天池云原生编程比赛的项目，使用语言为Go，修改的模块为Scheduler，特此记录。赛题详细说明请移步https://tianchi.aliyun.com/competition/entrance/231793/introduction

复赛描述：一个简化的Faas系统分为APIServer，Scheduler，ResourceManager，NodeService，ContainerService 5个组件，本题目中APIServer，ResourceManager，NodeService，ContainerService由平台提供，Scheduler的AcquireContainer和ReturnContainer API由选手实现（gRPC服务，语言不限），Scheduler会以容器方式单实例运行，无需考虑分布式多实例问题。
项目地址：https://github.com/YiDongMing/mini-faas
比赛过程及收获：比赛中可以深入研究的方向很多，但是如何设计自己的程序处理逻辑能在赛题的模拟数据下取得最好的成绩是关键点。在程序中实现了对容器的动态伸缩控制，然后使用不同的策略来对不同的请求进行容器的分配。
在复赛中使用的是go语言，第一次用go做项目，也使用了自己之前没有用过的protobuf协议，学习并使用新东西是很有趣的。比赛过程中在一个月内去学习新的技术栈并运用到比赛中是一个很有成就感的事情。最终排行榜第47名。
比赛涉及的技术框架：springboot、Java、Docker、Goland、protobuf通信协议

