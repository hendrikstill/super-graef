FROM scratch

COPY ./super-graef /super-graef
ENTRYPOINT ["/super-graef"]