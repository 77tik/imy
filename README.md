## 项目优点：
### 改写gentool：
+ 由于gorm的gentool过于幽默，tinyint和int 即使是unsigned 也会被gentool识别为int32，白白损失了一半的精度
    + 所以imygen改写了gentool，解决了这个问题：
    + 将tinyint unsigned 映射为 uint8
    + int unsigned 映射为 uint32
    + bigint unsigned 映射为 uint64
+ 优化gentool查询：
  + gorm查询过于复杂，所以imygen插件封装了常用的gorm查询
  + 使用：
    + imygen生成的代码位于dao目录下方，model内为结构和初始crud代码
    + 而.dao.go 则是优化后的查询方法
    + 如何使用？ dao.{表结构}.WithContext(l.ctx) 或 dao.{表结构}.{常用方法}

### 改写go-zero的handler生成
+ 改写parseHttp解析请求：

### 雪花ID
+ 雪花算法可以生成具有时间顺序性的唯一ID，方便系统对日志记录进行排序和查询。 
+ 在事件流处理系统中，事件需要按照时间顺序进行处理。 
+ 雪花算法生成的ID具有时间顺序性，可以作为事件的唯一标识和时间戳，方便系统对事件进行排序和处理

### 会话数据存储
+ 好友关系建立后会创建会话id，存入会话表
+ 群建立以后也会创建会话id，存入会话表
+ 每个会话id对应一个imy-timeline，这是一个持久化结构，专门存数据的，自带时序，顺序，用于做全量存储
+ 每个用户id也会对应一个imy-timeline，用于做同步存储
+ 当用户A发送一条消息给用户B时，首先会往会话ID：Conv_AB 对应的 imy-timeline写入一条信息；然后还会往双方的同步存储imy-timeline中写入信息
+ 群聊，则会往群会话ID对应的timeline写入信息，然后写入N个用户的同步库
+ 同步库：
  + 每个用户都有自己的同步库，这里放入了所有消息，比如AB之间的消息，AC之间的消息，A加入的所有群聊的消息
  + 用户登陆会自动拉去自己同步库中checkpoint之后的所有消息，然后这些消息，找到其对应的convId，记录将其对应的会话的自己的未读消息
+ 存储库：
  + 每个会话创建之后都会有自己对应的存储库，这个库是用来封存该会话下的所有信息的，平时一般提供历史消息的拉取
+ 客户端拉取同步信息以后，全部遍历一遍，会对每个会话 记录未读数；
+ 会话中的信息都是从存储库拉取的，
+ 流程是：
  + 1.A发送消息给B：服务端将消息写入会话-tl，A-tl，B-tl，ws通知B
  + 2.B收到ws通知，B客户端自动发起处理 B-tl中checkpoint之后的信息的请求，服务端遍历B-tl，把信息按会话ID分类，然后修改未读数，返回给B所有
    + 会话信息的未读数量，以及每个会话最新的一条消息的简略标识，以及同步库的最新checkpoint（其实就是序列号）
  + 3.此时B客户端终于看到了有多少信息是未读的，以及哪些会话是未读的，他点进其中一个会话，此时会发起拉取历史消息的请求
    + 服务端从存储库中返回最近的20条消息给客户端B
### imy-timeline存储
``` 
// 按照对齐原则，Test 变量占用 16 字节的内存
type Test struct {
    F1 uint64
    F2 uint32
    F3 byte
}
```
+ 但是如果我们使用json序列化，16字节会暴涨到30字节，足足相当于涨了一倍
+ 那么如何让这个结构体，序列化以后就是16字节呢？
``` 
func (t *Test) Marshal() ([]byte, error) {
   // 创建一个 16 字节的 buffer
   buf := make([]byte, 16)
   // 序列化
   binary.BigEndian.PutUint64(buf[0:8], t.F1) // BigEndian.PutUint64 是把一个64位的整型转换为一个大端序的数组
   binary.BigEndian.PutUint32(buf[8:12], t.F2)
   buf[12] = t.F3

   return buf, nil
}

func (t *Test) Unmarshal(buf []byte) error {
   if len(buf) != 16 {
      return errors.New("length not match")
   }
   // 反序列化
   t.F1 = binary.BigEndian.Uint64(buf[0:8])
   t.F2 = binary.BigEndian.Uint32(buf[8:12])
   t.F3 = buf[12]
   return nil
}
```
+ 什么是大端小端？
  + 大端小端只有在遇到多字节基础类型才会用到
  + 现在有一个uint32 的整数： 0x11 22 33 44，计算机会怎么存储呢？计算机存储的基本单位是字节，所以字节的摆放顺序决定了大小端
  + 大端：0x11 0x22 0x33 0x44 可以看到这就是人类的正常顺序
  + 小端：0x44 0x33 0x22 0x11 这个顺序就比较拟人了，所以人还是大最好
  + 从左到右 低地址 => 高地址，最高有效位在低地址就是大端，比如1000，决定大小的肯定是这个1对吧，所以他在左边
  + 大多数机器操作系统都是小端序，比如x86