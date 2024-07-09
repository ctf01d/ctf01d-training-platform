/**
 * CTF Management API
 * API for managing CTF (Capture The Flag) games, teams, users, and services.
 *
 * The version of the OpenAPI document: 1.0.0
 * 
 *
 * NOTE: This class is auto generated by OpenAPI Generator (https://openapi-generator.tech).
 * https://openapi-generator.tech
 * Do not edit the class manually.
 *
 */

import ApiClient from '../ApiClient';

/**
 * The TeamRequest model module.
 * @module model/TeamRequest
 * @version 1.0.0
 */
class TeamRequest {
    /**
     * Constructs a new <code>TeamRequest</code>.
     * @alias module:model/TeamRequest
     * @param name {String} Name of the team
     * @param universityId {Number} University or institution the team is associated with
     */
    constructor(name, universityId) { 
        
        TeamRequest.initialize(this, name, universityId);
    }

    /**
     * Initializes the fields of this object.
     * This method is used by the constructors of any subclasses, in order to implement multiple inheritance (mix-ins).
     * Only for internal use.
     */
    static initialize(obj, name, universityId) { 
        obj['name'] = name;
        obj['university_id'] = universityId;
    }

    /**
     * Constructs a <code>TeamRequest</code> from a plain JavaScript object, optionally creating a new instance.
     * Copies all relevant properties from <code>data</code> to <code>obj</code> if supplied or a new instance if not.
     * @param {Object} data The plain JavaScript object bearing properties of interest.
     * @param {module:model/TeamRequest} obj Optional instance to populate.
     * @return {module:model/TeamRequest} The populated <code>TeamRequest</code> instance.
     */
    static constructFromObject(data, obj) {
        if (data) {
            obj = obj || new TeamRequest();

            if (data.hasOwnProperty('name')) {
                obj['name'] = ApiClient.convertToType(data['name'], 'String');
            }
            if (data.hasOwnProperty('description')) {
                obj['description'] = ApiClient.convertToType(data['description'], 'String');
            }
            if (data.hasOwnProperty('university_id')) {
                obj['university_id'] = ApiClient.convertToType(data['university_id'], 'Number');
            }
            if (data.hasOwnProperty('social_links')) {
                obj['social_links'] = ApiClient.convertToType(data['social_links'], 'String');
            }
            if (data.hasOwnProperty('avatar_url')) {
                obj['avatar_url'] = ApiClient.convertToType(data['avatar_url'], 'String');
            }
        }
        return obj;
    }

    /**
     * Validates the JSON data with respect to <code>TeamRequest</code>.
     * @param {Object} data The plain JavaScript object bearing properties of interest.
     * @return {boolean} to indicate whether the JSON data is valid with respect to <code>TeamRequest</code>.
     */
    static validateJSON(data) {
        // check to make sure all required properties are present in the JSON string
        for (const property of TeamRequest.RequiredProperties) {
            if (!data.hasOwnProperty(property)) {
                throw new Error("The required field `" + property + "` is not found in the JSON data: " + JSON.stringify(data));
            }
        }
        // ensure the json data is a string
        if (data['name'] && !(typeof data['name'] === 'string' || data['name'] instanceof String)) {
            throw new Error("Expected the field `name` to be a primitive type in the JSON string but got " + data['name']);
        }
        // ensure the json data is a string
        if (data['description'] && !(typeof data['description'] === 'string' || data['description'] instanceof String)) {
            throw new Error("Expected the field `description` to be a primitive type in the JSON string but got " + data['description']);
        }
        // ensure the json data is a string
        if (data['social_links'] && !(typeof data['social_links'] === 'string' || data['social_links'] instanceof String)) {
            throw new Error("Expected the field `social_links` to be a primitive type in the JSON string but got " + data['social_links']);
        }
        // ensure the json data is a string
        if (data['avatar_url'] && !(typeof data['avatar_url'] === 'string' || data['avatar_url'] instanceof String)) {
            throw new Error("Expected the field `avatar_url` to be a primitive type in the JSON string but got " + data['avatar_url']);
        }

        return true;
    }


}

TeamRequest.RequiredProperties = ["name", "university_id"];

/**
 * Name of the team
 * @member {String} name
 */
TeamRequest.prototype['name'] = undefined;

/**
 * A brief description of the team
 * @member {String} description
 */
TeamRequest.prototype['description'] = undefined;

/**
 * University or institution the team is associated with
 * @member {Number} university_id
 */
TeamRequest.prototype['university_id'] = undefined;

/**
 * JSON string containing social media links of the team
 * @member {String} social_links
 */
TeamRequest.prototype['social_links'] = undefined;

/**
 * URL to the team's avatar
 * @member {String} avatar_url
 */
TeamRequest.prototype['avatar_url'] = undefined;






export default TeamRequest;

