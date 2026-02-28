fn main() -> Result<(), Box<dyn std::error::Error>> {
    tonic_build::configure()
        .build_server(false) // We only need the client
        .compile_protos(
            &["proto/streaming.proto", "proto/orderbook.proto"],
            &["proto"],
        )?;
    Ok(())
}
