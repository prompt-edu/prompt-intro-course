import { introCourseAxiosInstance } from '../introCourseServerConfig'
import { CreateKeycloakGroup } from '../../interfaces/CreateKeycloakGroup'

export const createCustomKeycloakGroup = async (
  courseID: string,
  group: CreateKeycloakGroup,
): Promise<void> => {
  try {
    await introCourseAxiosInstance.put(`/api/keycloak/${courseID}/group`, group, {
      headers: {
        'Content-Type': 'application/json-path+json',
      },
    })
  } catch (err) {
    console.error(err)
    throw err
  }
}
