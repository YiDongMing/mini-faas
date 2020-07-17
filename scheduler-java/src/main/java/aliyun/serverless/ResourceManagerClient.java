package aliyun.serverless;

import java.util.logging.Logger;
import java.util.concurrent.TimeUnit;

import io.grpc.Channel;
import io.grpc.ManagedChannel;
import io.grpc.ManagedChannelBuilder;
import io.grpc.StatusRuntimeException;

import resourcemanagerproto.ResourceManagerGrpc;
import resourcemanagerproto.ResourceManagerGrpc.*;
import resourcemanagerproto.ResourceManagerOuterClass.*;


public class ResourceManagerClient {

    private static final Logger logger = Logger.getLogger(ResourceManagerClient.class.getName());

    private final ResourceManagerBlockingStub blockingStub;

    public ResourceManagerClient(Channel channel) {
        blockingStub = ResourceManagerGrpc.newBlockingStub(channel);
    }

    public ReserveNodeReply reserveNode(ReserveNodeRequest req) {
        return blockingStub.reserveNode(req);
    }

    public ReleaseNodeReply releaseNode(ReleaseNodeRequest req) {
        return blockingStub.releaseNode(req);
    }

    public GetNodesUsageReply getNodesUsage(GetNodesUsageRequest req) {
        return blockingStub.getNodesUsage(req);
    }

    public static ResourceManagerClient New() {
        var rmEndpoint = System.getenv("RESOURCE_MANAGER_ENDPOINT");
        if (null == rmEndpoint) {
            rmEndpoint = "0.0.0.0:10400";
        }
        var channel = ManagedChannelBuilder.forTarget(rmEndpoint).usePlaintext().build();
        var client = new ResourceManagerClient(channel);
        logger.info("Connected to ResourceManager server at " + rmEndpoint);

        Runtime.getRuntime().addShutdownHook(new Thread(() -> {
            try {
                channel.shutdownNow().awaitTermination(5, TimeUnit.SECONDS);
            } catch (InterruptedException e) {
                e.printStackTrace(System.err);
            }
            System.err.println("ResourceManager client shut down.");
        }));

        return client;
    }
}
