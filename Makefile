run-server:
	@go run server/*.go

run-client:
	@cd client && npm run dev

gen-proto:
	@protoc --proto_path=proto proto/*.proto --go_out=proto --go-grpc_out=proto

gen-client:
	@protoc  --proto_path=proto proto/auth.proto \
					--js_out=import_style=commonjs:client/proto_gen \
					--grpc-web_out=import_style=typescript,mode=grpcwebtext:client/proto_gen




# protoc  --proto_path=proto proto/auth.proto \
# 					--js_out=import_style=commonjs,binary:client/proto_gen \
# 					--grpc-web_out=import_style=commonjs,mode=grpcwebtext:client/proto_gen

# npx protoc --ts_out proto_gen/  --proto_path ../proto ../proto/auth.proto