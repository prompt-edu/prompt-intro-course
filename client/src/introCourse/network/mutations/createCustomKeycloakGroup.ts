import type { CreateKeycloakGroup } from '../../interfaces/CreateKeycloakGroup'
import { introCourseAxiosInstance } from '../introCourseServerConfig'

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
