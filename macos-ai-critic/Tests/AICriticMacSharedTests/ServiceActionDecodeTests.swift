import XCTest
@testable import AICriticMacShared

/// Proves start/stop/restart server body shapes decode after HTTP 200
/// (required `message` would throw and skip refreshServices).
final class ServiceActionDecodeTests: XCTestCase {
    func testStartBody_serviceStatus_withoutMessage_decodes() throws {
        // Server Start handler encodes *ServiceStatus (no message field).
        let json = """
        {
          "id": "svc-1",
          "name": "demo",
          "status": "running",
          "pid": 42,
          "logPath": "/tmp/demo.log",
          "desiredRunning": true,
          "enabled": true
        }
        """.data(using: .utf8)!

        // Old required-message model would fail:
        struct Strict: Decodable {
            let status: String
            let message: String
        }
        XCTAssertThrowsError(try JSONDecoder().decode(Strict.self, from: json))

        let resp = try ServiceClient.decodeServiceActionBody(json)
        XCTAssertNotNil(resp)
        // start path must not throw — refreshServices can run after success
    }

    func testStopRestartBody_statusOk_withoutMessage_decodes() throws {
        let json = #"{"status":"ok"}"#.data(using: .utf8)!
        struct Strict: Decodable {
            let status: String
            let message: String
        }
        XCTAssertThrowsError(try JSONDecoder().decode(Strict.self, from: json))

        let resp = try ServiceClient.decodeServiceActionBody(json)
        XCTAssertEqual(resp.status, "ok")
        XCTAssertNil(resp.message)
    }

    func testEnableBody_withMessage_decodes() throws {
        let json = """
        {
          "status": "ok",
          "message": "The server won't start immediately until daemon checks at next time",
          "service": {
            "id": "svc-1",
            "name": "demo",
            "status": "stopped",
            "pid": 0,
            "logPath": "/tmp/demo.log",
            "desiredRunning": false,
            "enabled": true
          }
        }
        """.data(using: .utf8)!
        let resp = try ServiceClient.decodeServiceActionBody(json)
        XCTAssertEqual(resp.message, "The server won't start immediately until daemon checks at next time")
        XCTAssertEqual(resp.displayMessage, resp.message)
    }

    func testOptionalMessage_onServiceActionResponse() throws {
        let json = #"{"status":"ok"}"#.data(using: .utf8)!
        let resp = try JSONDecoder().decode(ServiceActionResponse.self, from: json)
        XCTAssertEqual(resp.status, "ok")
        XCTAssertNil(resp.message)
    }
}
