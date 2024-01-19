from detection.proto import queueService_pb2_grpc


class RouteGuideServicer(queueService_pb2_grpc.QueueServiceServicer):

    def NextFrame(self, request, context):

