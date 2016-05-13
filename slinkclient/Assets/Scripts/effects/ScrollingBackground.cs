using UnityEngine;
using System.Collections;

public class ScrollingBackground : MonoBehaviour {

    public Camera mainCam;

    const float PLANE_DEFAULT_SIZE = 10;
    const float TEXTURE_SIZE = 2048;

    // Use this for initialization
    void Start () {
	
	}
	
	// Update is called once per frame
	void Update () {
        float screenAspect = (float)Screen.width / (float)Screen.height;
        float cameraHeight = this.mainCam.orthographicSize * 2;
        float cameraWidth = cameraHeight * screenAspect;

        float yOffset = mainCam.transform.position.x / 11000f;
        float xOffset = mainCam.transform.position.y / 11000f;
        float xScale = cameraHeight / PLANE_DEFAULT_SIZE;
        float yScale = cameraWidth / PLANE_DEFAULT_SIZE;
        Material backMat = this.GetComponent<Renderer>().material;
        this.transform.localScale = new Vector3(xScale, 1, yScale);
        backMat.mainTextureScale = new Vector2(cameraHeight / TEXTURE_SIZE / 4f, cameraWidth / TEXTURE_SIZE / 4f);
        backMat.SetTextureOffset("_MainTex", new Vector2(xOffset, yOffset * -1));
    }
}
