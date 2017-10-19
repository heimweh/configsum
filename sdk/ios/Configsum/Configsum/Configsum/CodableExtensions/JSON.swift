//
//  JSON.swift
//  Configsum
//
//  Created by Michel Tabari on 2017-10-17.
//  From https://github.com/zoul/generic-json-swift

import Foundation

public enum JSON {
    case string(String)
    case number(Float)
    case object([String:JSON])
    case array([JSON])
    case bool(Bool)
    case null
}

public enum JSONError: Swift.Error {
    case decodingError
}

extension JSON: Equatable {
    
    public static func == (lhs: JSON, rhs: JSON) -> Bool {
        switch (lhs, rhs) {
        case (.string(let s1), .string(let s2)):
            return s1 == s2
        case (.number(let n1), .number(let n2)):
            return n1 == n2
        case (.object(let o1), .object(let o2)):
            return o1 == o2
        case (.array(let a1), .array(let a2)):
            return a1 == a2
        case (.bool(let b1), .bool(let b2)):
            return b1 == b2
        case (.null, .null):
            return true
        default:
            return false
        }
    }
}

extension JSON: CustomDebugStringConvertible {
    
    public var debugDescription: String {
        switch self {
        case .string(let str):
            return str.debugDescription
        case .number(let num):
            return num.debugDescription
        case .bool(let bool):
            return bool.description
        case .null:
            return "null"
        default:
            let encoder = JSONEncoder()
            encoder.outputFormatting = [.prettyPrinted]
            return try! String(data: encoder.encode(self), encoding: .utf8)!
        }
    }
}
