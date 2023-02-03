# 根据项目名称调整APP名称，建议和repo名称保持一致
ARG APP=p2cserver
# 运行阶段
FROM reg.smvm.cn/appbase/golang-build:1.17.8-alpine3.15 as final

RUN ls -l
ENV APP=${APP}
ENV WORKDIR=/data
COPY bin ${WORKDIR}/
WORKDIR ${WORKDIR}
CMD ["sh", "-c", "./${APP}"]