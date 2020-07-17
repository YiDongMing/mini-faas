package aliyun.serverless;

import org.testng.annotations.Test;

import resourcemanagerproto.ResourceManagerOuterClass.*;


public class ResourceManagerClientTest {

     @Test
     public void testNodeOperation() {
        // Remember to start ResourceManager server before run this case.
        var client = ResourceManagerClient.New();

        var reserveReq = ReserveNodeRequest.newBuilder()
                                           .setRequestId("test-request")
                                           .setAccountId("test-account")
                                           .build();
        var reserveRes = client.reserveNode(reserveReq);
        System.out.println(reserveRes);

        var releaseReq = ReleaseNodeRequest.newBuilder()
                                           .setRequestId("test-request")
                                           .setId(reserveRes.getNode().getId())
                                           .build();
        var releaseRes = client.releaseNode(releaseReq);
        System.out.println(releaseRes);
    }
}
